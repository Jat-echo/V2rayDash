package ssh

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	gossh "golang.org/x/crypto/ssh"
)

func testSSHServer(t *testing.T, password string, forwarding bool) (string, int, string) {
	t.Helper()
	_, private, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	signer, err := gossh.NewSignerFromKey(private)
	require.NoError(t, err)
	config := &gossh.ServerConfig{PasswordCallback: func(_ gossh.ConnMetadata, supplied []byte) (*gossh.Permissions, error) {
		if string(supplied) != password {
			return nil, fmt.Errorf("wrong password")
		}
		return nil, nil
	}}
	config.AddHostKey(signer)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	var mu sync.Mutex
	var connections []net.Conn
	t.Cleanup(func() {
		listener.Close()
		mu.Lock()
		defer mu.Unlock()
		for _, conn := range connections {
			conn.Close()
		}
	})
	go func() {
		for {
			raw, err := listener.Accept()
			if err != nil {
				return
			}
			mu.Lock()
			connections = append(connections, raw)
			mu.Unlock()
			go func() {
				defer raw.Close()
				raw.SetDeadline(time.Now().Add(5 * time.Second))
				connection, channels, requests, err := gossh.NewServerConn(raw, config)
				if err != nil {
					return
				}
				raw.SetDeadline(time.Time{})
				defer connection.Close()
				go gossh.DiscardRequests(requests)
				for incoming := range channels {
					if incoming.ChannelType() == "direct-tcpip" && forwarding {
						var target struct {
							Host       string
							Port       uint32
							Origin     string
							OriginPort uint32
						}
						if gossh.Unmarshal(incoming.ExtraData(), &target) != nil || target.Host != "127.0.0.1" {
							incoming.Reject(gossh.Prohibited, "invalid test target")
							continue
						}
						remote, err := net.DialTimeout("tcp", net.JoinHostPort(target.Host, fmt.Sprint(target.Port)), time.Second)
						if err != nil {
							incoming.Reject(gossh.ConnectionFailed, "unavailable")
							continue
						}
						channel, requests, err := incoming.Accept()
						if err != nil {
							remote.Close()
							continue
						}
						go gossh.DiscardRequests(requests)
						go func() { io.Copy(remote, channel); remote.Close() }()
						go func() { io.Copy(channel, remote); channel.Close() }()
					} else if incoming.ChannelType() == "session" && !forwarding {
						channel, requests, err := incoming.Accept()
						if err != nil {
							continue
						}
						go func() {
							defer channel.Close()
							for request := range requests {
								if request.Type != "exec" {
									request.Reply(false, nil)
									continue
								}
								request.Reply(true, nil)
								channel.Write([]byte("target-ok"))
								channel.SendRequest("exit-status", false, gossh.Marshal(struct{ Status uint32 }{0}))
								return
							}
						}()
					} else {
						incoming.Reject(gossh.UnknownChannelType, "unsupported")
					}
				}
			}()
		}
	}()
	host, port, err := net.SplitHostPort(listener.Addr().String())
	require.NoError(t, err)
	number, err := strconv.Atoi(port)
	require.NoError(t, err)
	return host, number, base64.StdEncoding.EncodeToString(signer.PublicKey().Marshal())
}

func TestSSHJumpUsesTargetAuthAndPinsBothHosts(t *testing.T) {
	host, port, key := testSSHServer(t, "target-secret", false)
	jumpHost, jumpPort, jumpKey := testSSHServer(t, "jump-secret", true)
	jump := &JumpHost{Host: jumpHost, Port: jumpPort, User: "jump", Auth: &PasswordAuth{Password: "jump-secret"}, HostKey: jumpKey}
	client, err := NewTrustedSSHClientViaJump(host, port, "target", &PasswordAuth{Password: "target-secret"}, key, jump)
	require.NoError(t, err)
	var out bytes.Buffer
	require.NoError(t, client.Execute("test-command", &out, io.Discard))
	require.Equal(t, "target-ok", out.String())
	require.Equal(t, key, client.HostKey)
	require.NoError(t, client.Close())
	_, err = NewTrustedSSHClientViaJump(host, port, "target", &PasswordAuth{Password: "wrong"}, key, jump)
	require.Error(t, err)
	_, err = NewTrustedSSHClientViaJump(host, port, "target", &PasswordAuth{Password: "target-secret"}, "wrong-host-key", jump)
	require.ErrorContains(t, err, "SSH host key changed")
	jump.HostKey = "wrong-jump-key"
	_, err = NewTrustedSSHClientViaJump(host, port, "target", &PasswordAuth{Password: "target-secret"}, key, jump)
	require.ErrorContains(t, err, "SSH host key changed")
	direct, err := NewTrustedSSHClient(host, port, "target", &PasswordAuth{Password: "target-secret"}, key)
	require.NoError(t, err)
	require.NoError(t, direct.Close())
}
