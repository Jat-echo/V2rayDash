package service

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"v2ray-dash/backend/internal/model"
	"v2ray-dash/backend/internal/repository"
	"v2ray-dash/backend/internal/ssh"
)

type RealityConfig struct {
	Enabled    bool
	ServerName string
	PublicKey  string
	Port       int
}

type AccountService struct {
	accountRepo *repository.AccountRepository
	serverRepo  *repository.ServerRepository
}

func NewAccountService(accountRepo *repository.AccountRepository, serverRepo *repository.ServerRepository) *AccountService {
	return &AccountService{
		accountRepo: accountRepo,
		serverRepo:  serverRepo,
	}
}

// GetAccountLink 生成单个账号的订阅链接
func (s *AccountService) GetAccountLink(account *model.Account, serverIP string, subType string, reality *RealityConfig) string {
	port := 443
	if reality != nil && reality.Enabled && reality.Port > 0 {
		port = reality.Port
	}

	var link string
	switch subType {
	case "vless":
		if reality != nil && reality.Enabled {
			// Reality 格式
			link = fmt.Sprintf("vless://%s@%s:%d?encryption=none&security=reality&sni=%s&fp=chrome&pbk=%s&sid=6ba85179e30d4fc2&flow=xtls-rprx-vision#%s",
				account.UUID, serverIP, port, reality.ServerName, reality.PublicKey, account.Email)
		} else {
			// 普通 TLS 格式
			link = fmt.Sprintf("vless://%s@%s:%d?encryption=none&flow=xtls-rprx-vision&security=tls&sni=%s#%s",
				account.UUID, serverIP, port, serverIP, account.Email)
		}
	case "ss":
		// ShadowRocket: 生成 SS URI (兼容格式)
		if reality != nil && reality.Enabled {
			// VLESS over SS (ShadowRocket 兼容)
			// 格式: ss://base64(method:password@host:port)#name
			// ShadowRocket 支持 vmess_protocol 等
			password := fmt.Sprintf("%s@%s:%d", account.UUID, serverIP, port)
			method := "chacha20-ietf-poly1305"
			encoded := base64.StdEncoding.EncodeToString([]byte(method + ":" + password))
			link = fmt.Sprintf("ss://%s#%s", encoded, account.Email)
		}
	case "clash_meta":
		// Placeholder link for Clash Meta - not used for subscription generation
		link = fmt.Sprintf("clash://%s@%s:7890", account.UUID, serverIP)
	default:
		link = fmt.Sprintf("vless://%s@%s:%d", account.UUID, serverIP, port)
	}
	return link
}

// GenerateVLESSSubscription 生成 VLESS 订阅内容
func (s *AccountService) GenerateVLESSSubscription(accounts []*model.Account, serverIP string, reality *RealityConfig) string {
	var lines []string
	for _, acc := range accounts {
		if !acc.Enabled {
			continue
		}
		// Generate VLESS URIs
		link := s.GetAccountLink(acc, serverIP, "vless", reality)
		lines = append(lines, link)
	}
	return strings.Join(lines, "\n")
}

// GenerateClashMetaSubscription 生成 Clash.Meta 订阅内容
func (s *AccountService) GenerateClashMetaSubscription(accounts []*model.Account, serverIP string, reality *RealityConfig, serverName string) (string, error) {
	port := 443
	if reality != nil && reality.Enabled && reality.Port > 0 {
		port = reality.Port
	}

	// 使用服务器名称作为节点名称
	nodeName := serverName
	if nodeName == "" {
		nodeName = "Proxy"
	}

	proxies := make([]map[string]interface{}, 0)
	for _, acc := range accounts {
		if !acc.Enabled {
			continue
		}

		var proxy map[string]interface{}
		if reality != nil && reality.Enabled {
			proxy = map[string]interface{}{
				"name":                   nodeName,
				"type":                   "vless",
				"server":                 serverIP,
				"port":                   port,
				"uuid":                   acc.UUID,
				"network":                "tcp",
				"tls":                    true,
				"udp":                    true,
				"flow":                   "xtls-rprx-vision",
				"servername":             reality.ServerName,
				"client-fingerprint":     "chrome",
				"reality-opts": map[string]interface{}{
					"public-key": reality.PublicKey,
					"short-id":   "6ba85179e30d4fc2",
				},
			}
		} else {
			proxy = map[string]interface{}{
				"name":               nodeName,
				"type":               "vless",
				"server":             serverIP,
				"port":               port,
				"uuid":               acc.UUID,
				"network":            "tcp",
				"tls":                true,
				"udp":                true,
				"flow":               "xtls-rprx-vision",
				"client-fingerprint": "chrome",
			}
		}
		proxies = append(proxies, proxy)
	}

	// 构建代理组名称列表，使用服务器名称
	proxyNames := []string{nodeName}

	// 使用用户提供的模板格式
	config := map[string]interface{}{
		"port":                     7890,
		"allow-lan":                true,
		"log-level":                "info",
		"external-controller":      "0.0.0.0:9090",
		"dns": map[string]interface{}{
			"enabled":          true,
			"listen":           "0.0.0.0:1053",
			"ipv6":              true,
			"enhanced-mode":    "fake-ip",
			"fake-ip-range":    "198.18.0.1/16",
			"fake-ip-filter": []string{
				"*.lan",
				"*.linksys.com",
				"*.linksyssmartwifi.com",
				"swscan.apple.com",
				"mesu.apple.com",
				"*.msftconnecttest.com",
				"*.msftncsi.com",
				"time.*.com",
				"time.*.gov",
				"time.*.edu.cn",
				"time.*.apple.com",
				"time1.*.com",
				"time2.*.com",
				"time3.*.com",
				"time4.*.com",
				"time5.*.com",
				"time6.*.com",
				"time7.*.com",
				"ntp.*.com",
				"ntp1.*.com",
				"ntp2.*.com",
				"ntp3.*.com",
				"ntp4.*.com",
				"ntp5.*.com",
				"ntp6.*.com",
				"ntp7.*.com",
				"*.time.edu.cn",
				"*.ntp.org.cn",
				"+.pool.ntp.org",
				"time1.cloud.tencent.com",
				"+.music.163.com",
				"*.126.net",
				"musicapi.taihe.com",
				"music.taihe.com",
				"songsearch.kugou.com",
				"trackercdn.kugou.com",
				"*.kuwo.cn",
				"api-jooxtt.sanook.com",
				"api.joox.com",
				"joox.com",
				"+.y.qq.com",
				"+.music.tc.qq.com",
				"aqqmusic.tc.qq.com",
				"+.stream.qqmusic.qq.com",
				"*.xiami.com",
				"+.music.migu.cn",
				"+.srv.nintendo.net",
				"+.stun.playstation.net",
				"xbox.*.microsoft.com",
				"+.xboxlive.com",
				"localhost.ptlogin2.qq.com",
				"proxy.golang.org",
				"stun.*.*",
				"stun.*.*.*",
				"*.mcdn.bilivideo.cn",
			},
			"default-nameserver": []string{
				"223.5.5.5",
				"114.114.114.114",
			},
			"nameserver": []string{
				"https://doh.pub/dns-query",
				"https://dns.alidns.com/dns-query",
			},
			"fallback-filter": map[string]interface{}{
				"geoip": false,
				"ipcidr": []string{
					"240.0.0.0/4",
					"0.0.0.0/32",
				},
			},
		},
		"proxies":       proxies,
		"proxy-groups": buildProxyGroups(proxyNames),
		"rules":        buildClashRules(),
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// buildProxyGroups 构建代理组
func buildProxyGroups(proxyNames []string) []map[string]interface{} {
	if len(proxyNames) == 0 {
		proxyNames = []string{"DIRECT"}
	}

	groups := []map[string]interface{}{
		{
			"name": "PROXY",
			"type": "select",
			"proxies": append([]string{"URLTest AutoSelect", "Fallback"}, proxyNames...),
		},
		{
			"name":     "URLTest AutoSelect",
			"type":     "url-test",
			"proxies":  proxyNames,
			"url":      "http://www.gstatic.com/generate_204",
			"interval": 300,
			"tolerance": 50,
		},
		{
			"name":     "Fallback",
			"type":     "fallback",
			"proxies":  proxyNames,
			"url":      "http://www.gstatic.com/generate_204",
			"interval": 300,
		},
		{
			"name":    "Final",
			"type":    "select",
			"proxies": []string{"PROXY", "DIRECT"},
		},
		{
			"name":    "Domestic",
			"type":    "select",
			"proxies": []string{"DIRECT", "PROXY"},
		},
		{
			"name":    "Apple",
			"type":    "select",
			"proxies": []string{"DIRECT", "PROXY"},
		},
		{
			"name":    "GlobalMedia",
			"type":    "select",
			"proxies": append([]string{"PROXY", "DIRECT"}, proxyNames...),
		},
		{
			"name":    "HKMTMedia",
			"type":    "select",
			"proxies": append([]string{"DIRECT", "PROXY"}, proxyNames...),
		},
		{
			"name":    "Ads",
			"type":    "select",
			"proxies": []string{"REJECT", "DIRECT"},
		},
		{
			"name":    "SPEEDTEST",
			"type":    "select",
			"proxies": append([]string{"PROXY"}, proxyNames...),
		},
		{
			"name":    "WHOER",
			"type":    "select",
			"proxies": append([]string{"PROXY"}, proxyNames...),
		},
		{
			"name":    "TT",
			"type":    "select",
			"proxies": append([]string{"PROXY"}, proxyNames...),
		},
		{
			"name":    "AI",
			"type":    "select",
			"proxies": proxyNames,
		},
	}

	return groups
}

// buildClashRules 构建分流规则（完整版）
func buildClashRules() []string {
	return []string{
		// Google
		"DOMAIN-SUFFIX,figma.com,PROXY",
		"DOMAIN-SUFFIX,paypalobjects.com,PROXY",
		"DOMAIN-SUFFIX,paypal.com,PROXY",
		"DOMAIN-SUFFIX,bard.google.com,PROXY",
		"DOMAIN-SUFFIX,colamanhua.com,PROXY",
		"DOMAIN-SUFFIX,appspot.com,PROXY",
		"DOMAIN-SUFFIX,blogger.com,PROXY",
		"DOMAIN-SUFFIX,getoutline.org,PROXY",
		"DOMAIN-SUFFIX,gvt0.com,PROXY",
		"DOMAIN-SUFFIX,gvt1.com,PROXY",
		"DOMAIN-SUFFIX,gvt3.com,PROXY",
		"DOMAIN-SUFFIX,xn--ngstr-lra8j.com,PROXY",
		"DOMAIN-KEYWORD,google,PROXY",
		"DOMAIN-KEYWORD,blogspot,PROXY",
		"DOMAIN-SUFFIX,onedrive.live.com,PROXY",
		"DOMAIN-SUFFIX,xboxlive.com,PROXY",
		"DOMAIN-SUFFIX,google.com,PROXY",
		"DOMAIN-SUFFIX,googleapis.com,PROXY",
		"DOMAIN-SUFFIX,googleadservices.com,PROXY",
		"DOMAIN-SUFFIX,googlevideo.com,PROXY",
		"DOMAIN-SUFFIX,googleusercontent.com,PROXY",
		"DOMAIN-SUFFIX,googletraveladservices.com,PROXY",
		"DOMAIN-SUFFIX,doubleclick.net,PROXY",
		"DOMAIN-SUFFIX,youtube.com,PROXY",
		"DOMAIN-SUFFIX,ytimg.com,PROXY",
		"DOMAIN-SUFFIX,1drv.com,PROXY",
		"DOMAIN-SUFFIX,1drv.ms,PROXY",
		"DOMAIN-SUFFIX,blob.core.windows.net,PROXY",
		"DOMAIN-SUFFIX,livefilestore.com,PROXY",
		"DOMAIN-SUFFIX,onedrive.com,PROXY",
		"DOMAIN-SUFFIX,storage.live.com,PROXY",
		"DOMAIN-SUFFIX,storage.msn.com,PROXY",
		"DOMAIN,oneclient.sfx.ms,PROXY",
		"DOMAIN-SUFFIX,gstatic.com,PROXY",
		"DOMAIN-SUFFIX,gmail.com,PROXY",
		"DOMAIN-SUFFIX,yt.be,PROXY",
		"DOMAIN-SUFFIX,youtu.be,PROXY",
		"DOMAIN-SUFFIX,ggpht.com,PROXY",
		"DOMAIN-SUFFIX,googleusercontent.com,PROXY",
		"DOMAIN-SUFFIX,admob.com,PROXY",
		"DOMAIN-SUFFIX,doubleclick.com,PROXY",
		"DOMAIN-SUFFIX,fls.doubleclick.net,PROXY",
		"DOMAIN-SUFFIX,metrics.google.com,PROXY",
		"DOMAIN-SUFFIX,plus.google.com,PROXY",
		"DOMAIN-SUFFIX,accounts.google.com,PROXY",
		// Facebook
		"DOMAIN-SUFFIX,facebook.com,PROXY",
		"DOMAIN-SUFFIX,fbcdn.net,PROXY",
		"DOMAIN-SUFFIX,instagram.com,PROXY",
		"DOMAIN-SUFFIX,m.me,PROXY",
		"DOMAIN-SUFFIX,messenger.com,PROXY",
		"DOMAIN-SUFFIX,oculus.com,PROXY",
		"DOMAIN-SUFFIX,oculuscdn.com,PROXY",
		"DOMAIN-SUFFIX,rocksdb.org,PROXY",
		"DOMAIN-SUFFIX,whatsapp.com,PROXY",
		"DOMAIN-SUFFIX,whatsapp.net,PROXY",
		"DOMAIN-KEYWORD,facebook,PROXY",
		"DOMAIN-SUFFIX,fb.com,PROXY",
		"DOMAIN-SUFFIX,fb.me,PROXY",
		"DOMAIN-SUFFIX,fbaddins.com,PROXY",
		"DOMAIN-SUFFIX,fbsbx.com,PROXY",
		"DOMAIN-SUFFIX,fbworkmail.com,PROXY",
		"DOMAIN-SUFFIX,cdninstagram.com,PROXY",
		"DOMAIN-SUFFIX,pscp.tv,PROXY",
		"DOMAIN-SUFFIX,periscope.tv,PROXY",
		// Twitter/X
		"DOMAIN-SUFFIX,twitter.com,PROXY",
		"DOMAIN-SUFFIX,twimg.com,PROXY",
		"DOMAIN-SUFFIX,t.co,PROXY",
		"DOMAIN-SUFFIX,twimg.co,PROXY",
		"DOMAIN-SUFFIX,twitpic.com,PROXY",
		"DOMAIN-SUFFIX,vine.co,PROXY",
		"DOMAIN-KEYWORD,twitter,PROXY",
		// Telegram
		"DOMAIN-SUFFIX,t.me,PROXY",
		"DOMAIN-SUFFIX,tdesktop.com,PROXY",
		"DOMAIN-SUFFIX,telegra.ph,PROXY",
		"DOMAIN-SUFFIX,telegram.me,PROXY",
		"DOMAIN-SUFFIX,telegram.org,PROXY",
		"DOMAIN-SUFFIX,telegram.com,PROXY",
		// AI
		"DOMAIN-SUFFIX,bing.com,AI",
		"DOMAIN-SUFFIX,claude.ai,AI",
		"DOMAIN-SUFFIX,openai.com,AI",
		"DOMAIN-SUFFIX,chatgpt.com,AI",
		"DOMAIN-SUFFIX,anthropic.com,AI",
		"DOMAIN-SUFFIX,nexusmedia-ua.com,PROXY",
		// Netflix
		"DOMAIN-KEYWORD,netflix,GlobalMedia",
		"DOMAIN,netflix.com.edgesuite.net,GlobalMedia",
		"DOMAIN-SUFFIX,netflix.com,GlobalMedia",
		"DOMAIN-SUFFIX,netflix.com.edgesuite.net,GlobalMedia",
		"DOMAIN-SUFFIX,netflix.net,GlobalMedia",
		"DOMAIN-SUFFIX,netflixdnstest0.com,GlobalMedia",
		"DOMAIN-SUFFIX,netflixdnstest1.com,GlobalMedia",
		"DOMAIN-SUFFIX,netflixdnstest2.com,GlobalMedia",
		"DOMAIN-SUFFIX,netflixdnstest3.com,GlobalMedia",
		"DOMAIN-SUFFIX,netflixdnstest4.com,GlobalMedia",
		"DOMAIN-SUFFIX,netflixdnstest5.com,GlobalMedia",
		"DOMAIN-SUFFIX,netflixdnstest6.com,GlobalMedia",
		"DOMAIN-SUFFIX,netflixdnstest7.com,GlobalMedia",
		"DOMAIN-SUFFIX,netflixdnstest8.com,GlobalMedia",
		"DOMAIN-SUFFIX,nflxext.com,GlobalMedia",
		"DOMAIN-SUFFIX,nflximg.com,GlobalMedia",
		"DOMAIN-SUFFIX,nflximg.net,GlobalMedia",
		"DOMAIN-SUFFIX,nflxso.net,GlobalMedia",
		"DOMAIN-SUFFIX,nflxvideo.net,GlobalMedia",
		"DOMAIN-SUFFIX,netflix.com,GlobalMedia",
		"DOMAIN-SUFFIX,nflxvideo.net,GlobalMedia",
		// YouTube
		"DOMAIN-SUFFIX,youtube.com,PROXY",
		"DOMAIN-SUFFIX,googlevideo.com,PROXY",
		"DOMAIN-SUFFIX,ytimg.com,PROXY",
		"DOMAIN,youtubei.googleapis.com,PROXY",
		// TikTok
		"DOMAIN,p16-tiktokcdn-com.akamaized.net,PROXY",
		"DOMAIN-SUFFIX,amemv.com,PROXY",
		"DOMAIN-SUFFIX,byteoversea.com,TT",
		"DOMAIN-SUFFIX,ibytedtos.com,TT",
		"DOMAIN-SUFFIX,ibyteimg.com,TT",
		"DOMAIN-SUFFIX,ipstatp.com,TT",
		"DOMAIN-SUFFIX,muscdn.com,TT",
		"DOMAIN-SUFFIX,musical.ly,TT",
		"DOMAIN-SUFFIX,sgpstatp.com,TT",
		"DOMAIN-SUFFIX,snssdk.com,TT",
		"DOMAIN-SUFFIX,tik-tokapi.com,TT",
		"DOMAIN-SUFFIX,tiktok.com,TT",
		"DOMAIN-SUFFIX,tiktokcdn.com,TT",
		"DOMAIN-SUFFIX,tiktokv.com,TT",
		"DOMAIN-KEYWORD,-tiktokcdn-com,TT",
		// Speedtest
		"DOMAIN-SUFFIX,speedtest.net,SPEEDTEST",
		"DOMAIN-SUFFIX,whoer.net,WHOER",
		"DOMAIN-SUFFIX,euromonitor.com,PROXY",
		"DOMAIN-SUFFIX,v2ex.com,PROXY",
		// Apple
		"DOMAIN-SUFFIX,aaplimg.com,Apple",
		"DOMAIN-SUFFIX,apple.co,Apple",
		"DOMAIN-SUFFIX,apple.com,Apple",
		"DOMAIN-SUFFIX,appstore.com,Apple",
		"DOMAIN-SUFFIX,cdn-apple.com,Apple",
		"DOMAIN-SUFFIX,crashlytics.com,Apple",
		"DOMAIN-SUFFIX,icloud.com,Apple",
		"DOMAIN-SUFFIX,icloud-content.com,Apple",
		"DOMAIN-SUFFIX,me.com,Apple",
		"DOMAIN-SUFFIX,mzstatic.com,Apple",
		"DOMAIN,www-cdn.icloud.com.akadns.net,Apple",
		"DOMAIN-SUFFIX,apple.com,Apple",
		"DOMAIN-SUFFIX,icloud.com,Apple",
		"DOMAIN,testflight.apple.com,PROXY",
		"DOMAIN-SUFFIX,appsto.re,PROXY",
		"DOMAIN,books.itunes.apple.com,PROXY",
		"DOMAIN,hls.itunes.apple.com,PROXY",
		"DOMAIN,apps.apple.com,PROXY",
		"DOMAIN,itunes.apple.com,PROXY",
		"DOMAIN,api-glb-sea.smoot.apple.com,PROXY",
		"DOMAIN,lookup-api.apple.com,PROXY",
		"IP-CIDR,17.0.0.0/8,Apple",
		// GlobalMedia
		"DOMAIN-SUFFIX,deezer.com,GlobalMedia",
		"DOMAIN-SUFFIX,dzcdn.net,GlobalMedia",
		"DOMAIN-SUFFIX,kkbox.com,GlobalMedia",
		"DOMAIN-SUFFIX,kkbox.com.tw,GlobalMedia",
		"DOMAIN-SUFFIX,kfs.io,GlobalMedia",
		"DOMAIN-SUFFIX,joox.com,GlobalMedia",
		"DOMAIN-SUFFIX,pandora.com,GlobalMedia",
		"DOMAIN-SUFFIX,p-cdn.us,GlobalMedia",
		"DOMAIN-SUFFIX,sndcdn.com,GlobalMedia",
		"DOMAIN-SUFFIX,soundcloud.com,GlobalMedia",
		"DOMAIN-SUFFIX,pscdn.co,GlobalMedia",
		"DOMAIN-SUFFIX,scdn.co,GlobalMedia",
		"DOMAIN-SUFFIX,spotify.com,GlobalMedia",
		"DOMAIN-SUFFIX,spoti.fi,GlobalMedia",
		"DOMAIN-SUFFIX,tidal.com,GlobalMedia",
		"DOMAIN-SUFFIX,c4assets.com,GlobalMedia",
		"DOMAIN-SUFFIX,channel4.com,GlobalMedia",
		"DOMAIN-SUFFIX,abema.io,GlobalMedia",
		"DOMAIN-SUFFIX,abema.tv,GlobalMedia",
		"DOMAIN-SUFFIX,ameba.jp,GlobalMedia",
		"DOMAIN-SUFFIX,hayabusa.io,GlobalMedia",
		"DOMAIN,abematv.akamaized.net,GlobalMedia",
		"DOMAIN,ds-linear-abematv.akamaized.net,GlobalMedia",
		"DOMAIN,ds-vod-abematv.akamaized.net,GlobalMedia",
		"DOMAIN,linear-abematv.akamaized.net,GlobalMedia",
		"DOMAIN-SUFFIX,aiv-cdn.net,GlobalMedia",
		"DOMAIN-SUFFIX,aiv-delivery.net,GlobalMedia",
		"DOMAIN-SUFFIX,amazonvideo.com,GlobalMedia",
		"DOMAIN-SUFFIX,llnwd.net,GlobalMedia",
		"DOMAIN-SUFFIX,media-amazon.com,GlobalMedia",
		"DOMAIN-SUFFIX,primevideo.com,GlobalMedia",
		"DOMAIN-SUFFIX,bahamut.com.tw,GlobalMedia",
		"DOMAIN-SUFFIX,gamer.com.tw,GlobalMedia",
		"DOMAIN,gamer-cds.cdn.hinet.net,GlobalMedia",
		"DOMAIN,gamer2-cds.cdn.hinet.net,GlobalMedia",
		"DOMAIN-SUFFIX,bbc.co.uk,GlobalMedia",
		"DOMAIN-SUFFIX,bbci.co.uk,GlobalMedia",
		"DOMAIN-KEYWORD,bbcfmt,GlobalMedia",
		"DOMAIN-KEYWORD,uk-live,GlobalMedia",
		"DOMAIN-SUFFIX,dazn.com,GlobalMedia",
		"DOMAIN-SUFFIX,encoretvb.com,GlobalMedia",
		"DOMAIN,edge.api.brightcove.com,GlobalMedia",
		"DOMAIN,bcbolt446c5271-a.akamaihd.net,GlobalMedia",
		"DOMAIN-SUFFIX,dashasiafox.akamaized.net,GlobalMedia",
		"DOMAIN-SUFFIX,fox.com,GlobalMedia",
		"DOMAIN-SUFFIX,foxdcg.com,GlobalMedia",
		"DOMAIN-SUFFIX,foxplus.com,GlobalMedia",
		"DOMAIN-SUFFIX,staticasiafox.akamaized.net,GlobalMedia",
		"DOMAIN-SUFFIX,theplatform.com,GlobalMedia",
		"DOMAIN-SUFFIX,uplynk.com,GlobalMedia",
		"DOMAIN-SUFFIX,hbo.com,GlobalMedia",
		"DOMAIN-SUFFIX,hbogo.com,GlobalMedia",
		"DOMAIN-SUFFIX,hboasia.com,GlobalMedia",
		"DOMAIN-SUFFIX,hbogoasia.hk,GlobalMedia",
		"DOMAIN,44wilhpljf.execute-api.ap-southeast-1.amazonaws.com,GlobalMedia",
		"DOMAIN,bcbolthboa-a.akamaihd.net,GlobalMedia",
		"DOMAIN,cf-images.ap-southeast-1.prod.boltdns.net,GlobalMedia",
		"DOMAIN,manifest.prod.boltdns.net,GlobalMedia",
		"DOMAIN,s3-ap-southeast-1.amazonaws.com,GlobalMedia",
		"DOMAIN-SUFFIX,5itv.tv,GlobalMedia",
		"DOMAIN-SUFFIX,ocnttv.com,GlobalMedia",
		"DOMAIN-SUFFIX,hulu.com,GlobalMedia",
		"DOMAIN-SUFFIX,huluim.com,GlobalMedia",
		"DOMAIN-SUFFIX,hulustream.com,GlobalMedia",
		"DOMAIN-SUFFIX,happyon.jp,GlobalMedia",
		"DOMAIN-SUFFIX,hulu.jp,GlobalMedia",
		"DOMAIN-SUFFIX,itv.com,GlobalMedia",
		"DOMAIN-SUFFIX,itvstatic.com,GlobalMedia",
		"DOMAIN,itvpnpmobile-a.akamaihd.net,GlobalMedia",
		"DOMAIN-SUFFIX,kktv.com.tw,GlobalMedia",
		"DOMAIN-SUFFIX,kktv.me,GlobalMedia",
		"DOMAIN,kktv-theater.kk.stream,GlobalMedia",
		"DOMAIN-SUFFIX,linetv.tw,GlobalMedia",
		"DOMAIN,d3c7rimkq79yfu.cloudfront.net,GlobalMedia",
		"DOMAIN-SUFFIX,litv.tv,GlobalMedia",
		"DOMAIN,litvfreemobile-hichannel.cdn.hinet.net,GlobalMedia",
		"DOMAIN-SUFFIX,channel5.com,GlobalMedia",
		"DOMAIN-SUFFIX,my5.tv,GlobalMedia",
		"DOMAIN,d349g9zuie06uo.cloudfront.net,GlobalMedia",
		"DOMAIN-SUFFIX,mytvsuper.com,GlobalMedia",
		"DOMAIN-SUFFIX,tvb.com,GlobalMedia",
		"DOMAIN-SUFFIX,dmc.nico,GlobalMedia",
		"DOMAIN-SUFFIX,nicovideo.jp,GlobalMedia",
		"DOMAIN-SUFFIX,nimg.jp,GlobalMedia",
		"DOMAIN-SUFFIX,socdm.com,GlobalMedia",
		"DOMAIN-SUFFIX,pbs.org,GlobalMedia",
		"DOMAIN-SUFFIX,phncdn.com,GlobalMedia",
		"DOMAIN-SUFFIX,pornhub.com,GlobalMedia",
		"DOMAIN-SUFFIX,pornhubpremium.com,GlobalMedia",
		"DOMAIN-SUFFIX,skyking.com.tw,GlobalMedia",
		"DOMAIN,hamifans.emome.net,GlobalMedia",
		"DOMAIN-SUFFIX,twitch.tv,GlobalMedia",
		"DOMAIN-SUFFIX,twitchcdn.net,GlobalMedia",
		"DOMAIN-SUFFIX,ttvnw.net,GlobalMedia",
		"DOMAIN-SUFFIX,viu.com,GlobalMedia",
		"DOMAIN-SUFFIX,viu.tv,GlobalMedia",
		"DOMAIN,api.viu.now.com,GlobalMedia",
		"DOMAIN,d1k2us671qcoau.cloudfront.net,GlobalMedia",
		"DOMAIN,d2anahhhmp1ffz.cloudfront.net,GlobalMedia",
		"DOMAIN,dfp6rglgjqszk.cloudfront.net,GlobalMedia",
		"DOMAIN-SUFFIX,googlevideo.com,GlobalMedia",
		"DOMAIN-SUFFIX,youtube.com,GlobalMedia",
		"DOMAIN-SUFFIX,dmc.nico,GlobalMedia",
		"DOMAIN-SUFFIX,nicovideo.jp,GlobalMedia",
		"DOMAIN-SUFFIX,hbo.com,GlobalMedia",
		"DOMAIN-SUFFIX,fox.com,GlobalMedia",
		"DOMAIN-SUFFIX,dazn.com,GlobalMedia",
		// HKMTMedia
		"DOMAIN-SUFFIX,iqiyi.com,HKMTMedia",
		"DOMAIN-SUFFIX,71.am,HKMTMedia",
		"DOMAIN-SUFFIX,bilibili.com,HKMTMedia",
		"DOMAIN,upos-hz-mirrorakam.akamaized.net,HKMTMedia",
		"DOMAIN-SUFFIX,bilibili.com,HKMTMedia",
		"DOMAIN-SUFFIX,iqiyi.com,HKMTMedia",
		"DOMAIN-SUFFIX,weibo.com,HKMTMedia",
		"DOMAIN-SUFFIX,viu.com,HKMTMedia",
		"DOMAIN-SUFFIX,viu.tv,HKMTMedia",
		// Domestic
		"DOMAIN-SUFFIX,douban.com,Domestic",
		"DOMAIN-SUFFIX,baidu.com,Domestic",
		"DOMAIN-SUFFIX,taobao.com,Domestic",
		"DOMAIN-SUFFIX,tmall.com,Domestic",
		"DOMAIN-SUFFIX,jd.com,Domestic",
		"DOMAIN-SUFFIX,qq.com,Domestic",
		"DOMAIN-SUFFIX,163.com,Domestic",
		"DOMAIN-SUFFIX,126.net,Domestic",
		"DOMAIN-SUFFIX,baidu.com,Domestic",
		"DOMAIN-SUFFIX,taobao.com,Domestic",
		"DOMAIN-SUFFIX,tmall.com,Domestic",
		"DOMAIN-SUFFIX,jd.com,Domestic",
		"DOMAIN-SUFFIX,qq.com,Domestic",
		"DOMAIN-SUFFIX,163.com,Domestic",
		"DOMAIN-SUFFIX,126.net,Domestic",
		"DOMAIN-SUFFIX,alibaba.com,Domestic",
		"DOMAIN-SUFFIX,alipay.com,Domestic",
		"DOMAIN-SUFFIX,amap.com,Domestic",
		"DOMAIN-SUFFIX,dingtalk.com,Domestic",
		"DOMAIN-SUFFIX,weibo.com,Domestic",
		"DOMAIN-SUFFIX,bilibili.com,Domestic",
		"DOMAIN-SUFFIX,youku.com,Domestic",
		"DOMAIN-SUFFIX,iqiyi.com,Domestic",
		"DOMAIN-SUFFIX,mgtv.com,Domestic",
		// Ads
		"DOMAIN-SUFFIX,17gouwuba.com,Ads",
		"DOMAIN-SUFFIX,186078.com,Ads",
		"DOMAIN-SUFFIX,189zj.cn,Ads",
		"DOMAIN-SUFFIX,285680.com,Ads",
		"DOMAIN-SUFFIX,3721zh.com,Ads",
		"DOMAIN-SUFFIX,4336wang.cn,Ads",
		"DOMAIN-SUFFIX,51chumoping.com,Ads",
		"DOMAIN-SUFFIX,51mld.cn,Ads",
		"DOMAIN-SUFFIX,51mypc.cn,Ads",
		"DOMAIN-SUFFIX,58mingri.cn,Ads",
		"DOMAIN-SUFFIX,58mingtian.cn,Ads",
		"DOMAIN-SUFFIX,5vl58stm.com,Ads",
		"DOMAIN-SUFFIX,6d63d3.com,Ads",
		"DOMAIN-SUFFIX,7gg.cc,Ads",
		"DOMAIN-SUFFIX,91veg.com,Ads",
		"DOMAIN-SUFFIX,9s6q.cn,Ads",
		"DOMAIN-SUFFIX,adsame.com,Ads",
		"DOMAIN-SUFFIX,aiclk.com,Ads",
		"DOMAIN-SUFFIX,akuai.top,Ads",
		"DOMAIN-SUFFIX,atplay.cn,Ads",
		"DOMAIN-SUFFIX,baiwanchuangyi.com,Ads",
		"DOMAIN-SUFFIX,beerto.cn,Ads",
		"DOMAIN-SUFFIX,beilamusi.com,Ads",
		"DOMAIN-SUFFIX,benshiw.net,Ads",
		"DOMAIN-SUFFIX,bianxianmao.com,Ads",
		"DOMAIN-SUFFIX,bryonypie.com,Ads",
		"DOMAIN-SUFFIX,cishantao.com,Ads",
		"DOMAIN-SUFFIX,cszlks.com,Ads",
		"DOMAIN-SUFFIX,cudaojia.com,Ads",
		"DOMAIN-SUFFIX,dafapromo.com,Ads",
		"DOMAIN-SUFFIX,daitdai.com,Ads",
		"DOMAIN-SUFFIX,dsaeerf.com,Ads",
		"DOMAIN-SUFFIX,dugesheying.com,Ads",
		"DOMAIN-SUFFIX,dv8c1t.cn,Ads",
		"DOMAIN-SUFFIX,echatu.com,Ads",
		"DOMAIN-SUFFIX,erdoscs.com,Ads",
		"DOMAIN-SUFFIX,fan-yong.com,Ads",
		"DOMAIN-SUFFIX,feih.com.cn,Ads",
		"DOMAIN-SUFFIX,fjlqqc.com,Ads",
		"DOMAIN-SUFFIX,fkku194.com,Ads",
		"DOMAIN-SUFFIX,freedrive.cn,Ads",
		"DOMAIN-SUFFIX,gclick.cn,Ads",
		"DOMAIN-SUFFIX,goufanli100.com,Ads",
		"DOMAIN-SUFFIX,goupaoerdai.com,Ads",
		"DOMAIN-SUFFIX,gouwubang.com,Ads",
		"DOMAIN-SUFFIX,gzxnlk.com,Ads",
		"DOMAIN-SUFFIX,haoshengtoys.com,Ads",
		"DOMAIN-SUFFIX,hyunke.com,Ads",
		"DOMAIN-SUFFIX,ichaosheng.com,Ads",
		"DOMAIN-SUFFIX,ishop789.com,Ads",
		"DOMAIN-SUFFIX,jdkic.com,Ads",
		"DOMAIN-SUFFIX,jiubuhua.com,Ads",
		"DOMAIN-SUFFIX,jwg365.cn,Ads",
		"DOMAIN-SUFFIX,kawo77.com,Ads",
		"DOMAIN-SUFFIX,kualianyingxiao.cn,Ads",
		"DOMAIN-SUFFIX,kumihua.com,Ads",
		"DOMAIN-SUFFIX,ltheanine.cn,Ads",
		"DOMAIN-SUFFIX,maipinshangmao.com,Ads",
		"DOMAIN-SUFFIX,minisplat.cn,Ads",
		"DOMAIN-SUFFIX,mkitgfs.com,Ads",
		"DOMAIN-SUFFIX,mlnbike.com,Ads",
		"DOMAIN-SUFFIX,mobjump.com,Ads",
		"DOMAIN-SUFFIX,nbkbgd.cn,Ads",
		"DOMAIN-SUFFIX,newapi.com,Ads",
		"DOMAIN-SUFFIX,pinzhitmall.com,Ads",
		"DOMAIN-SUFFIX,poppyta.com,Ads",
		"DOMAIN-SUFFIX,qianchuanghr.com,Ads",
		"DOMAIN-SUFFIX,qichexin.com,Ads",
		"DOMAIN-SUFFIX,qinchugudao.com,Ads",
		"DOMAIN-SUFFIX,quanliyouxi.cn,Ads",
		"DOMAIN-SUFFIX,qutaobi.com,Ads",
		"DOMAIN-SUFFIX,ry51w.cn,Ads",
		"DOMAIN-SUFFIX,sg536.cn,Ads",
		"DOMAIN-SUFFIX,sifubo.cn,Ads",
		"DOMAIN-SUFFIX,sifuce.cn,Ads",
		"DOMAIN-SUFFIX,sifuda.cn,Ads",
		"DOMAIN-SUFFIX,sifufu.cn,Ads",
		"DOMAIN-SUFFIX,sifuge.cn,Ads",
		"DOMAIN-SUFFIX,sifugu.cn,Ads",
		"DOMAIN-SUFFIX,sifuhe.cn,Ads",
		"DOMAIN-SUFFIX,sifuhu.cn,Ads",
		"DOMAIN-SUFFIX,sifuji.cn,Ads",
		"DOMAIN-SUFFIX,sifuka.cn,Ads",
		"DOMAIN-SUFFIX,smgru.net,Ads",
		"DOMAIN-SUFFIX,taoggou.com,Ads",
		"DOMAIN-SUFFIX,tcxshop.com,Ads",
		"DOMAIN-SUFFIX,tjqonline.cn,Ads",
		"DOMAIN-SUFFIX,topitme.com,Ads",
		"DOMAIN-SUFFIX,tt3sm4.cn,Ads",
		"DOMAIN-SUFFIX,tuia.cn,Ads",
		"DOMAIN-SUFFIX,tuipenguin.com,Ads",
		"DOMAIN-SUFFIX,tuitiger.com,Ads",
		"DOMAIN-SUFFIX,websd8.com,Ads",
		"DOMAIN-SUFFIX,wx16999.com,Ads",
		"DOMAIN-SUFFIX,xiaohuau.xyz,Ads",
		"DOMAIN-SUFFIX,yinmong.com,Ads",
		"DOMAIN-SUFFIX,yiqifa.com,Ads",
		"DOMAIN-SUFFIX,yitaopt.com,Ads",
		"DOMAIN-SUFFIX,yjqiqi.com,Ads",
		"DOMAIN-SUFFIX,yukhj.com,Ads",
		"DOMAIN-SUFFIX,zhaozecheng.cn,Ads",
		"DOMAIN-SUFFIX,zhenxinet.com,Ads",
		"DOMAIN-SUFFIX,zlne800.com,Ads",
		"DOMAIN-SUFFIX,zunmi.cn,Ads",
		"DOMAIN-SUFFIX,zzd6.com,Ads",
		"DOMAIN-SUFFIX,kuaizip.com,Ads",
		"DOMAIN-SUFFIX,mackeeper.com,Ads",
		"DOMAIN-SUFFIX,flash.cn,Ads",
		"DOMAIN,geo2.adobe.com,Ads",
		"DOMAIN-SUFFIX,4009997658.com,Ads",
		"DOMAIN-SUFFIX,abbyychina.com,Ads",
		"DOMAIN-SUFFIX,bartender.cc,Ads",
		"DOMAIN-SUFFIX,betterzip.net,Ads",
		"DOMAIN-SUFFIX,beyondcompare.cc,Ads",
		"DOMAIN-SUFFIX,bingdianhuanyuan.cn,Ads",
		"DOMAIN-SUFFIX,chemdraw.com.cn,Ads",
		"DOMAIN-SUFFIX,cjmakeding.com,Ads",
		"DOMAIN-SUFFIX,cjmkt.com,Ads",
		"DOMAIN-SUFFIX,codesoftchina.com,Ads",
		"DOMAIN-SUFFIX,coreldrawchina.com,Ads",
		"DOMAIN-SUFFIX,crossoverchina.com,Ads",
		"DOMAIN-SUFFIX,dongmansoft.com,Ads",
		"DOMAIN-SUFFIX,earmasterchina.cn,Ads",
		"DOMAIN-SUFFIX,easyrecoverychina.com,Ads",
		"DOMAIN-SUFFIX,ediuschina.com,Ads",
		"DOMAIN-SUFFIX,flstudiochina.com,Ads",
		"DOMAIN-SUFFIX,formysql.com,Ads",
		"DOMAIN-SUFFIX,guitarpro.cc,Ads",
		"DOMAIN-SUFFIX,huishenghuiying.com.cn,Ads",
		"DOMAIN-SUFFIX,hypersnap.net,Ads",
		"DOMAIN-SUFFIX,iconworkshop.cn,Ads",
		"DOMAIN-SUFFIX,imindmap.cc,Ads",
		"DOMAIN-SUFFIX,jihehuaban.com.cn,Ads",
		"DOMAIN-SUFFIX,keyshot.cc,Ads",
		"DOMAIN-SUFFIX,kingdeecn.cn,Ads",
		"DOMAIN-SUFFIX,logoshejishi.com,Ads",
		"DOMAIN-SUFFIX,luping.net.cn,Ads",
		"DOMAIN-SUFFIX,mairuan.cn,Ads",
		"DOMAIN-SUFFIX,mairuan.com,Ads",
		"DOMAIN-SUFFIX,mairuan.com.cn,Ads",
		"DOMAIN-SUFFIX,mairuan.net,Ads",
		"DOMAIN-SUFFIX,mairuanwang.com,Ads",
		"DOMAIN-SUFFIX,makeding.com,Ads",
		"DOMAIN-SUFFIX,mathtype.cn,Ads",
		"DOMAIN-SUFFIX,mindmanager.cc,Ads",
		"DOMAIN-SUFFIX,mindmanager.cn,Ads",
		"DOMAIN-SUFFIX,mindmapper.cc,Ads",
		"DOMAIN-SUFFIX,mycleanmymac.com,Ads",
		"DOMAIN-SUFFIX,nicelabel.cc,Ads",
		"DOMAIN-SUFFIX,ntfsformac.cc,Ads",
		"DOMAIN-SUFFIX,ntfsformac.cn,Ads",
		"DOMAIN-SUFFIX,overturechina.com,Ads",
		"DOMAIN-SUFFIX,passwordrecovery.cn,Ads",
		"DOMAIN-SUFFIX,pdfexpert.cc,Ads",
		"DOMAIN-SUFFIX,photozoomchina.com,Ads",
		"DOMAIN-SUFFIX,shankejingling.com,Ads",
		"DOMAIN-SUFFIX,ultraiso.net,Ads",
		"DOMAIN-SUFFIX,vegaschina.cn,Ads",
		"DOMAIN-SUFFIX,xmindchina.net,Ads",
		"DOMAIN-SUFFIX,xshellcn.com,Ads",
		"DOMAIN-SUFFIX,yihuifu.cn,Ads",
		"DOMAIN-SUFFIX,yuanchengxiezuo.com,Ads",
		"DOMAIN-SUFFIX,zbrushcn.com,Ads",
		"DOMAIN-SUFFIX,zhzzx.com,Ads",
		"DOMAIN-SUFFIX,39.107.15.115/32,Ads",
		"DOMAIN-SUFFIX,47.89.59.182/32,Ads",
		"DOMAIN-SUFFIX,103.49.209.27/32,Ads",
		"DOMAIN-SUFFIX,123.56.152.96/32,Ads",
		"DOMAIN-SUFFIX,61.160.200.223/32,Ads",
		"DOMAIN-SUFFIX,61.160.200.242/32,Ads",
		"DOMAIN-SUFFIX,61.160.200.252/32,Ads",
		"DOMAIN-SUFFIX,61.174.50.214/32,Ads",
		"DOMAIN-SUFFIX,111.175.220.163/32,Ads",
		"DOMAIN-SUFFIX,111.175.220.164/32,Ads",
		"DOMAIN-SUFFIX,124.232.160.178/32,Ads",
		"DOMAIN-SUFFIX,175.6.223.15/32,Ads",
		"DOMAIN-SUFFIX,183.59.53.237/32,Ads",
		"DOMAIN-SUFFIX,218.93.127.37/32,Ads",
		"DOMAIN-SUFFIX,221.228.17.152/32,Ads",
		"DOMAIN-SUFFIX,221.231.6.79/32,Ads",
		"DOMAIN-SUFFIX,222.186.61.91/32,Ads",
		"DOMAIN-SUFFIX,222.186.61.95/32,Ads",
		"DOMAIN-SUFFIX,222.186.61.96/32,Ads",
		"DOMAIN-SUFFIX,222.186.61.97/32,Ads",
		"DOMAIN-SUFFIX,106.75.231.48/32,Ads",
		"DOMAIN-SUFFIX,119.4.249.166/32,Ads",
		"DOMAIN-SUFFIX,220.196.52.141/32,Ads",
		"DOMAIN-SUFFIX,221.6.4.148/32,Ads",
		"DOMAIN-SUFFIX,114.247.28.96/32,Ads",
		"DOMAIN-SUFFIX,221.179.131.72/32,Ads",
		"DOMAIN-SUFFIX,221.179.140.145/32,Ads",
		"DOMAIN-SUFFIX,10.72.25.0/24,Ads",
		"DOMAIN-SUFFIX,115.182.16.79/32,Ads",
		"DOMAIN-SUFFIX,118.144.88.126/32,Ads",
		"DOMAIN-SUFFIX,118.144.88.215/32,Ads",
		"DOMAIN-SUFFIX,118.144.88.216/32,Ads",
		"DOMAIN-SUFFIX,120.76.189.132/32,Ads",
		"DOMAIN-SUFFIX,124.14.21.147/32,Ads",
		"DOMAIN-SUFFIX,124.14.21.151/32,Ads",
		"DOMAIN-SUFFIX,180.166.52.24/32,Ads",
		"DOMAIN-SUFFIX,211.161.101.106/32,Ads",
		"DOMAIN-SUFFIX,220.115.251.25/32,Ads",
		"DOMAIN-SUFFIX,222.73.156.235/32,Ads",
		// Direct rules
		"DOMAIN-SUFFIX,googletraveladservices.com,DIRECT",
		"DOMAIN,dl.google.com,DIRECT",
		"DOMAIN,mtalk.google.com,DIRECT",
		"DOMAIN-SUFFIX,qhres.com,DIRECT",
		"DOMAIN-SUFFIX,qhimg.com,DIRECT",
		"DOMAIN-SUFFIX,akadns.net,DIRECT",
		"DOMAIN-SUFFIX,alibaba.com,DIRECT",
		"DOMAIN-SUFFIX,alicdn.com,DIRECT",
		"DOMAIN-SUFFIX,alikunlun.com,DIRECT",
		"DOMAIN-SUFFIX,alipay.com,DIRECT",
		"DOMAIN-SUFFIX,amap.com,DIRECT",
		"DOMAIN-SUFFIX,autonavi.com,DIRECT",
		"DOMAIN-SUFFIX,dingtalk.com,DIRECT",
		"DOMAIN-SUFFIX,mxhichina.com,DIRECT",
		"DOMAIN-SUFFIX,soku.com,DIRECT",
		"DOMAIN-SUFFIX,taobao.com,DIRECT",
		"DOMAIN-SUFFIX,tmall.com,DIRECT",
		"DOMAIN-SUFFIX,tmall.hk,DIRECT",
		"DOMAIN-SUFFIX,ykimg.com,DIRECT",
		"DOMAIN-SUFFIX,youku.com,DIRECT",
		"DOMAIN-SUFFIX,xiami.com,DIRECT",
		"DOMAIN-SUFFIX,xiami.net,DIRECT",
		"DOMAIN-SUFFIX,baidu.com,DIRECT",
		"DOMAIN-SUFFIX,baidubcr.com,DIRECT",
		"DOMAIN-SUFFIX,bdstatic.com,DIRECT",
		"DOMAIN-SUFFIX,yunjiasu-cdn.net,DIRECT",
		"DOMAIN-SUFFIX,acgvideo.com,DIRECT",
		"DOMAIN-SUFFIX,biliapi.com,DIRECT",
		"DOMAIN-SUFFIX,biliapi.net,DIRECT",
		"DOMAIN-SUFFIX,bilibili.com,DIRECT",
		"DOMAIN-SUFFIX,bilibili.tv,DIRECT",
		"DOMAIN-SUFFIX,hdslb.com,DIRECT",
		"DOMAIN-SUFFIX,blizzard.com,DIRECT",
		"DOMAIN-SUFFIX,battle.net,DIRECT",
		"DOMAIN,blzddist1-a.akamaihd.net,DIRECT",
		"DOMAIN-SUFFIX,feiliao.com,DIRECT",
		"DOMAIN-SUFFIX,pstatp.com,DIRECT",
		"DOMAIN-SUFFIX,snssdk.com,DIRECT",
		"DOMAIN-SUFFIX,iesdouyin.com,DIRECT",
		"DOMAIN-SUFFIX,toutiao.com,DIRECT",
		"DOMAIN-SUFFIX,cctv.com,DIRECT",
		"DOMAIN-SUFFIX,cctvpic.com,DIRECT",
		"DOMAIN-SUFFIX,livechina.com,DIRECT",
		"DOMAIN-SUFFIX,didialift.com,DIRECT",
		"DOMAIN-SUFFIX,didiglobal.com,DIRECT",
		"DOMAIN-SUFFIX,udache.com,DIRECT",
		"DOMAIN-SUFFIX,343480.com,DIRECT",
		"DOMAIN-SUFFIX,baduziyuan.com,DIRECT",
		"DOMAIN-SUFFIX,com-hs-hkdy.com,DIRECT",
		"DOMAIN-SUFFIX,czybjz.com,DIRECT",
		"DOMAIN-SUFFIX,dandanzan.com,DIRECT",
		"DOMAIN-SUFFIX,fjhps.com,DIRECT",
		"DOMAIN-SUFFIX,kuyunbo.club,DIRECT",
		"DOMAIN-SUFFIX,21cn.com,DIRECT",
		"DOMAIN-SUFFIX,hitv.com,DIRECT",
		"DOMAIN-SUFFIX,mgtv.com,DIRECT",
		"DOMAIN-SUFFIX,iqiyi.com,DIRECT",
		"DOMAIN-SUFFIX,iqiyipic.com,DIRECT",
		"DOMAIN-SUFFIX,71.am.com,DIRECT",
		"DOMAIN-SUFFIX,jd.com,DIRECT",
		"DOMAIN-SUFFIX,jd.hk,DIRECT",
		"DOMAIN-SUFFIX,jdpay.com,DIRECT",
		"DOMAIN-SUFFIX,360buyimg.com,DIRECT",
		"DOMAIN-SUFFIX,iciba.com,DIRECT",
		"DOMAIN-SUFFIX,ksosoft.com,DIRECT",
		"DOMAIN-SUFFIX,meitu.com,DIRECT",
		"DOMAIN-SUFFIX,meitudata.com,DIRECT",
		"DOMAIN-SUFFIX,meitustat.com,DIRECT",
		"DOMAIN-SUFFIX,meipai.com,DIRECT",
		"DOMAIN-SUFFIX,duokan.com,DIRECT",
		"DOMAIN-SUFFIX,mi-img.com,DIRECT",
		"DOMAIN-SUFFIX,miui.com,DIRECT",
		"DOMAIN-SUFFIX,miwifi.com,DIRECT",
		"DOMAIN-SUFFIX,xiaomi.com,DIRECT",
		"DOMAIN-SUFFIX,msecnd.net,DIRECT",
		"DOMAIN-SUFFIX,office365.com,DIRECT",
		"DOMAIN-SUFFIX,outlook.com,DIRECT",
		"DOMAIN-SUFFIX,visualstudio.com,DIRECT",
		"DOMAIN-SUFFIX,windows.com,DIRECT",
		"DOMAIN-SUFFIX,windowsupdate.com,DIRECT",
		"DOMAIN-SUFFIX,163.com,DIRECT",
		"DOMAIN-SUFFIX,126.net,DIRECT",
		"DOMAIN-SUFFIX,127.net,DIRECT",
		"DOMAIN-SUFFIX,163yun.com,DIRECT",
		"DOMAIN-SUFFIX,lofter.com,DIRECT",
		"DOMAIN-SUFFIX,netease.com,DIRECT",
		"DOMAIN-SUFFIX,ydstatic.com,DIRECT",
		"DOMAIN-SUFFIX,sina.com,DIRECT",
		"DOMAIN-SUFFIX,weibo.com,DIRECT",
		"DOMAIN-SUFFIX,weibocdn.com,DIRECT",
		"DOMAIN-SUFFIX,sohu.com,DIRECT",
		"DOMAIN-SUFFIX,sohucs.com,DIRECT",
		"DOMAIN-SUFFIX,sohu-inc.com,DIRECT",
		"DOMAIN-SUFFIX,v-56.com,DIRECT",
		"DOMAIN-SUFFIX,sogo.com,DIRECT",
		"DOMAIN-SUFFIX,sogou.com,DIRECT",
		"DOMAIN-SUFFIX,sogoucdn.com,DIRECT",
		"DOMAIN-SUFFIX,steampowered.com,DIRECT",
		"DOMAIN-SUFFIX,steam-chat.com,DIRECT",
		"DOMAIN-SUFFIX,steamgames.com,DIRECT",
		"DOMAIN-SUFFIX,steamusercontent.com,DIRECT",
		"DOMAIN-SUFFIX,steamcontent.com,DIRECT",
		"DOMAIN-SUFFIX,steamstatic.com,DIRECT",
		"DOMAIN-SUFFIX,steamcdn-a.akamaihd.net,DIRECT",
		"DOMAIN-SUFFIX,steamstat.us,DIRECT",
		"DOMAIN-SUFFIX,gtimg.com,DIRECT",
		"DOMAIN-SUFFIX,idqqimg.com,DIRECT",
		"DOMAIN-SUFFIX,igamecj.com,DIRECT",
		"DOMAIN-SUFFIX,myapp.com,DIRECT",
		"DOMAIN-SUFFIX,myqcloud.com,DIRECT",
		"DOMAIN-SUFFIX,qq.com,DIRECT",
		"DOMAIN-SUFFIX,tencent.com,DIRECT",
		"DOMAIN-SUFFIX,tencent-cloud.net,DIRECT",
		"DOMAIN-SUFFIX,jstucdn.com,DIRECT",
		"DOMAIN-SUFFIX,zimuzu.io,DIRECT",
		"DOMAIN-SUFFIX,zimuzu.tv,DIRECT",
		"DOMAIN-SUFFIX,zmz2019.com,DIRECT",
		"DOMAIN-SUFFIX,zmzapi.com,DIRECT",
		"DOMAIN-SUFFIX,zmzapi.net,DIRECT",
		"DOMAIN-SUFFIX,zmzfile.com,DIRECT",
		"DOMAIN-SUFFIX,ccgslb.com,DIRECT",
		"DOMAIN-SUFFIX,ccgslb.net,DIRECT",
		"DOMAIN-SUFFIX,chinanetcenter.com,DIRECT",
		"DOMAIN-SUFFIX,meixincdn.com,DIRECT",
		"DOMAIN-SUFFIX,ourdvs.com,DIRECT",
		"DOMAIN-SUFFIX,staticdn.net,DIRECT",
		"DOMAIN-SUFFIX,wangsu.com,DIRECT",
		"DOMAIN-SUFFIX,ipip.net,DIRECT",
		"DOMAIN-SUFFIX,ip.la,DIRECT",
		"DOMAIN-SUFFIX,ip-cdn.com,DIRECT",
		"DOMAIN-SUFFIX,ipv6-test.com,DIRECT",
		"DOMAIN-SUFFIX,test-ipv6.com,DIRECT",
		"DOMAIN-SUFFIX,whatismyip.com,DIRECT",
		"DOMAIN-SUFFIX,netspeedtestmaster.com,DIRECT",
		"DOMAIN,speedtest.macpaw.com,DIRECT",
		"DOMAIN-SUFFIX,awesome-hd.me,DIRECT",
		"DOMAIN-SUFFIX,broadcasthe.net,DIRECT",
		"DOMAIN-SUFFIX,chdbits.co,DIRECT",
		"DOMAIN-SUFFIX,classix-unlimited.co.uk,DIRECT",
		"DOMAIN-SUFFIX,empornium.me,DIRECT",
		"DOMAIN-SUFFIX,gazellegames.net,DIRECT",
		"DOMAIN-SUFFIX,hdchina.org,DIRECT",
		"DOMAIN-SUFFIX,hdsky.me,DIRECT",
		"DOMAIN-SUFFIX,jpopsuki.eu,DIRECT",
		"DOMAIN-SUFFIX,keepfrds.com,DIRECT",
		"DOMAIN-SUFFIX,m-team.cc,DIRECT",
		"DOMAIN-SUFFIX,nanyangpt.com,DIRECT",
		"DOMAIN-SUFFIX,ncore.cc,DIRECT",
		"DOMAIN-SUFFIX,open.cd,DIRECT",
		"DOMAIN-SUFFIX,ourbits.club,DIRECT",
		"DOMAIN-SUFFIX,passthepopcorn.me,DIRECT",
		"DOMAIN-SUFFIX,privatehd.to,DIRECT",
		"DOMAIN-SUFFIX,redacted.ch,DIRECT",
		"DOMAIN-SUFFIX,springsunday.net,DIRECT",
		"DOMAIN-SUFFIX,tjupt.org,DIRECT",
		"DOMAIN-SUFFIX,totheglory.im,DIRECT",
		"DOMAIN-SUFFIX,cn,DIRECT",
		"DOMAIN-SUFFIX,360in.com,DIRECT",
		"DOMAIN-SUFFIX,51ym.me,DIRECT",
		"DOMAIN-SUFFIX,8686c.com,DIRECT",
		"DOMAIN-SUFFIX,abchina.com,DIRECT",
		"DOMAIN-SUFFIX,accuweather.com,DIRECT",
		"DOMAIN-SUFFIX,aicoinstorge.com,DIRECT",
		"DOMAIN-SUFFIX,air-matters.com,DIRECT",
		"DOMAIN-SUFFIX,air-matters.io,DIRECT",
		"DOMAIN-SUFFIX,aixifan.com,DIRECT",
		"DOMAIN-SUFFIX,amd.com,DIRECT",
		"DOMAIN-SUFFIX,b612.net,DIRECT",
		"DOMAIN-SUFFIX,bdatu.com,DIRECT",
		"DOMAIN-SUFFIX,beitaichufang.com,DIRECT",
		"DOMAIN-SUFFIX,bjango.com,DIRECT",
		"DOMAIN-SUFFIX,booking.com,DIRECT",
		"DOMAIN-SUFFIX,bstatic.com,DIRECT",
		"DOMAIN-SUFFIX,cailianpress.com,DIRECT",
		"DOMAIN-SUFFIX,camera360.com,DIRECT",
		"DOMAIN-SUFFIX,chinaso.com,DIRECT",
		"DOMAIN-SUFFIX,chua.pro,DIRECT",
		"DOMAIN-SUFFIX,chuimg.com,DIRECT",
		"DOMAIN-SUFFIX,chunyu.mobi,DIRECT",
		"DOMAIN-SUFFIX,chushou.tv,DIRECT",
		"DOMAIN-SUFFIX,cmbchina.com,DIRECT",
		"DOMAIN-SUFFIX,cmbimg.com,DIRECT",
		"DOMAIN-SUFFIX,ctrip.com,DIRECT",
		"DOMAIN-SUFFIX,dfcfw.com,DIRECT",
		"DOMAIN-SUFFIX,docschina.org,DIRECT",
		"DOMAIN-SUFFIX,douban.com,DIRECT",
		"DOMAIN-SUFFIX,doubanio.com,DIRECT",
		"DOMAIN-SUFFIX,douyu.com,DIRECT",
		"DOMAIN-SUFFIX,dxycdn.com,DIRECT",
		"DOMAIN-SUFFIX,dytt8.net,DIRECT",
		"DOMAIN-SUFFIX,eastmoney.com,DIRECT",
		"DOMAIN-SUFFIX,eudic.net,DIRECT",
		"DOMAIN-SUFFIX,feng.com,DIRECT",
		"DOMAIN-SUFFIX,fengkongcloud.com,DIRECT",
		"DOMAIN-SUFFIX,frdic.com,DIRECT",
		"DOMAIN-SUFFIX,futu5.com,DIRECT",
		"DOMAIN-SUFFIX,futunn.com,DIRECT",
		"DOMAIN-SUFFIX,gandi.net,DIRECT",
		"DOMAIN-SUFFIX,geilicdn.com,DIRECT",
		"DOMAIN-SUFFIX,getpricetag.com,DIRECT",
		"DOMAIN-SUFFIX,gifshow.com,DIRECT",
		"DOMAIN-SUFFIX,godic.net,DIRECT",
		"DOMAIN-SUFFIX,hicloud.com,DIRECT",
		"DOMAIN-SUFFIX,hongxiu.com,DIRECT",
		"DOMAIN-SUFFIX,hostbuf.com,DIRECT",
		"DOMAIN-SUFFIX,huxiucdn.com,DIRECT",
		"DOMAIN-SUFFIX,huya.com,DIRECT",
		"DOMAIN-SUFFIX,infinitynewtab.com,DIRECT",
		"DOMAIN-SUFFIX,ithome.com,DIRECT",
		"DOMAIN-SUFFIX,java.com,DIRECT",
		"DOMAIN-SUFFIX,jidian.im,DIRECT",
		"DOMAIN-SUFFIX,kaiyanapp.com,DIRECT",
		"DOMAIN-SUFFIX,kaspersky-labs.com,DIRECT",
		"DOMAIN-SUFFIX,keepcdn.com,DIRECT",
		"DOMAIN-SUFFIX,kkmh.com,DIRECT",
		"DOMAIN-SUFFIX,licdn.com,DIRECT",
		"DOMAIN-SUFFIX,linkedin.com,DIRECT",
		"DOMAIN-SUFFIX,loli.net,DIRECT",
		"DOMAIN-SUFFIX,luojilab.com,DIRECT",
		"DOMAIN-SUFFIX,maoyan.com,DIRECT",
		"DOMAIN-SUFFIX,maoyun.tv,DIRECT",
		"DOMAIN-SUFFIX,meituan.com,DIRECT",
		"DOMAIN-SUFFIX,meituan.net,DIRECT",
		"DOMAIN-SUFFIX,mobike.com,DIRECT",
		"DOMAIN-SUFFIX,moke.com,DIRECT",
		"DOMAIN-SUFFIX,mubu.com,DIRECT",
		"DOMAIN-SUFFIX,myzaker.com,DIRECT",
		"DOMAIN-SUFFIX,nim-lang-cn.org,DIRECT",
		"DOMAIN-SUFFIX,nvidia.com,DIRECT",
		"DOMAIN-SUFFIX,oracle.com,DIRECT",
		"DOMAIN-SUFFIX,paypal.com,DIRECT",
		"DOMAIN-SUFFIX,paypalobjects.com,DIRECT",
		"DOMAIN-SUFFIX,qdaily.com,DIRECT",
		"DOMAIN-SUFFIX,qidian.com,DIRECT",
		"DOMAIN-SUFFIX,qyer.com,DIRECT",
		"DOMAIN-SUFFIX,qyerstatic.com,DIRECT",
		"DOMAIN-SUFFIX,raychase.net,DIRECT",
		"DOMAIN-SUFFIX,ronghub.com,DIRECT",
		"DOMAIN-SUFFIX,ruguoapp.com,DIRECT",
		"DOMAIN-SUFFIX,s-reader.com,DIRECT",
		"DOMAIN-SUFFIX,sankuai.com,DIRECT",
		"DOMAIN-SUFFIX,scomper.me,DIRECT",
		"DOMAIN-SUFFIX,seafile.com,DIRECT",
		"DOMAIN-SUFFIX,sm.ms,DIRECT",
		"DOMAIN-SUFFIX,smzdm.com,DIRECT",
		"DOMAIN-SUFFIX,snapdrop.net,DIRECT",
		"DOMAIN-SUFFIX,snwx.com,DIRECT",
		"DOMAIN-SUFFIX,sspai.com,DIRECT",
		"DOMAIN-SUFFIX,takungpao.com,DIRECT",
		"DOMAIN-SUFFIX,teamviewer.com,DIRECT",
		"DOMAIN-SUFFIX,tianyancha.com,DIRECT",
		"DOMAIN-SUFFIX,udacity.com,DIRECT",
		"DOMAIN-SUFFIX,uning.com,DIRECT",
		"DOMAIN-SUFFIX,vmware.com,DIRECT",
		"DOMAIN-SUFFIX,weather.com,DIRECT",
		"DOMAIN-SUFFIX,weico.cc,DIRECT",
		"DOMAIN-SUFFIX,weidian.com,DIRECT",
		"DOMAIN-SUFFIX,xiachufang.com,DIRECT",
		"DOMAIN-SUFFIX,ximalaya.com,DIRECT",
		"DOMAIN-SUFFIX,xinhuanet.com,DIRECT",
		"DOMAIN-SUFFIX,xmcdn.com,DIRECT",
		"DOMAIN-SUFFIX,yangkeduo.com,DIRECT",
		"DOMAIN-SUFFIX,zhangzishi.cc,DIRECT",
		"DOMAIN-SUFFIX,zhihu.com,DIRECT",
		"DOMAIN-SUFFIX,zhimg.com,DIRECT",
		"DOMAIN-SUFFIX,zhuihd.com,DIRECT",
		"DOMAIN,download.jetbrains.com,DIRECT",
		"DOMAIN,images-cn.ssl-images-amazon.com,DIRECT",
		// IP rules
		"IP-CIDR,127.0.0.0/8,DIRECT",
		"IP-CIDR,192.168.0.0/16,DIRECT",
		"IP-CIDR,10.0.0.0/8,DIRECT",
		"IP-CIDR,172.16.0.0/12,DIRECT",
		"IP-CIDR,100.64.0.0/10,DIRECT",
		"IP-CIDR,39.107.15.115/32,Ads",
		"IP-CIDR,47.89.59.182/32,Ads",
		"IP-CIDR,103.49.209.27/32,Ads",
		"IP-CIDR,123.56.152.96/32,Ads",
		"IP-CIDR,61.160.200.223/32,Ads",
		"IP-CIDR,61.160.200.242/32,Ads",
		"IP-CIDR,61.160.200.252/32,Ads",
		"IP-CIDR,61.174.50.214/32,Ads",
		"IP-CIDR,111.175.220.163/32,Ads",
		"IP-CIDR,111.175.220.164/32,Ads",
		"IP-CIDR,124.232.160.178/32,Ads",
		"IP-CIDR,175.6.223.15/32,Ads",
		"IP-CIDR,183.59.53.237/32,Ads",
		"IP-CIDR,218.93.127.37/32,Ads",
		"IP-CIDR,221.228.17.152/32,Ads",
		"IP-CIDR,221.231.6.79/32,Ads",
		"IP-CIDR,222.186.61.91/32,Ads",
		"IP-CIDR,222.186.61.95/32,Ads",
		"IP-CIDR,222.186.61.96/32,Ads",
		"IP-CIDR,222.186.61.97/32,Ads",
		"IP-CIDR,106.75.231.48/32,Ads",
		"IP-CIDR,119.4.249.166/32,Ads",
		"IP-CIDR,220.196.52.141/32,Ads",
		"IP-CIDR,221.6.4.148/32,Ads",
		"IP-CIDR,114.247.28.96/32,Ads",
		"IP-CIDR,221.179.131.72/32,Ads",
		"IP-CIDR,221.179.140.145/32,Ads",
		"IP-CIDR,10.72.25.0/24,Ads",
		"IP-CIDR,115.182.16.79/32,Ads",
		"IP-CIDR,118.144.88.126/32,Ads",
		"IP-CIDR,118.144.88.215/32,Ads",
		"IP-CIDR,118.144.88.216/32,Ads",
		"IP-CIDR,120.76.189.132/32,Ads",
		"IP-CIDR,124.14.21.147/32,Ads",
		"IP-CIDR,124.14.21.151/32,Ads",
		"IP-CIDR,180.166.52.24/32,Ads",
		"IP-CIDR,211.161.101.106/32,Ads",
		"IP-CIDR,220.115.251.25/32,Ads",
		"IP-CIDR,222.73.156.235/32,Ads",
		"IP-CIDR,35.190.247.0/24,PROXY",
		"IP-CIDR,64.233.160.0/19,PROXY",
		"IP-CIDR,66.102.0.0/20,PROXY",
		"IP-CIDR,66.249.80.0/20,PROXY",
		"IP-CIDR,72.14.192.0/18,PROXY",
		"IP-CIDR,74.125.0.0/16,PROXY",
		"IP-CIDR,108.177.8.0/21,PROXY",
		"IP-CIDR,172.217.0.0/16,PROXY",
		"IP-CIDR,173.194.0.0/16,PROXY",
		"IP-CIDR,209.85.128.0/17,PROXY",
		"IP-CIDR,216.58.192.0/19,PROXY",
		"IP-CIDR,216.239.32.0/19,PROXY",
		"IP-CIDR,31.13.24.0/21,PROXY",
		"IP-CIDR,31.13.64.0/18,PROXY",
		"IP-CIDR,45.64.40.0/22,PROXY",
		"IP-CIDR,66.220.144.0/20,PROXY",
		"IP-CIDR,69.63.176.0/20,PROXY",
		"IP-CIDR,69.171.224.0/19,PROXY",
		"IP-CIDR,74.119.76.0/22,PROXY",
		"IP-CIDR,103.4.96.0/22,PROXY",
		"IP-CIDR,129.134.0.0/17,PROXY",
		"IP-CIDR,157.240.0.0/17,PROXY",
		"IP-CIDR,173.252.64.0/19,PROXY",
		"IP-CIDR,173.252.96.0/19,PROXY",
		"IP-CIDR,179.60.192.0/22,PROXY",
		"IP-CIDR,185.60.216.0/22,PROXY",
		"IP-CIDR,204.15.20.0/22,PROXY",
		"IP-CIDR,3.123.36.126/32,PROXY",
		"IP-CIDR,35.157.215.84/32,PROXY",
		"IP-CIDR,35.157.217.255/32,PROXY",
		"IP-CIDR,52.58.209.134/32,PROXY",
		"IP-CIDR,54.93.124.31/32,PROXY",
		"IP-CIDR,54.162.243.80/32,PROXY",
		"IP-CIDR,54.173.34.141/32,PROXY",
		"IP-CIDR,54.235.23.242/32,PROXY",
		"IP-CIDR,169.45.248.118/32,PROXY",
		"IP-CIDR,69.195.160.0/19,PROXY",
		"IP-CIDR,104.244.42.0/21,PROXY",
		"IP-CIDR,192.133.76.0/22,PROXY",
		"IP-CIDR,199.16.156.0/22,PROXY",
		"IP-CIDR,199.59.148.0/22,PROXY",
		"IP-CIDR,199.96.56.0/21,PROXY",
		"IP-CIDR,202.160.128.0/22,PROXY",
		"IP-CIDR,209.237.192.0/19,PROXY",
		"IP-CIDR,67.198.55.0/24,PROXY",
		"IP-CIDR,91.108.4.0/22,PROXY",
		"IP-CIDR,91.108.8.0/22,PROXY",
		"IP-CIDR,91.108.12.0/22,PROXY",
		"IP-CIDR,91.108.16.0/22,PROXY",
		"IP-CIDR,91.108.56.0/22,PROXY",
		"IP-CIDR,109.239.140.0/24,PROXY",
		"IP-CIDR,149.154.160.0/20,PROXY",
		"IP-CIDR,205.172.60.0/22,PROXY",
		"IP-CIDR,103.2.30.0/23,PROXY",
		"IP-CIDR,125.209.208.0/20,PROXY",
		"IP-CIDR,147.92.128.0/17,PROXY",
		"IP-CIDR,203.104.144.0/21,PROXY",
		"IP-CIDR,23.246.0.0/18,GlobalMedia",
		"IP-CIDR,37.77.184.0/21,GlobalMedia",
		"IP-CIDR,45.57.0.0/17,GlobalMedia",
		"IP-CIDR,64.120.128.0/17,GlobalMedia",
		"IP-CIDR,66.197.128.0/17,GlobalMedia",
		"IP-CIDR,108.175.32.0/20,GlobalMedia",
		"IP-CIDR,192.173.64.0/18,GlobalMedia",
		"IP-CIDR,198.38.96.0/19,GlobalMedia",
		"IP-CIDR,198.45.48.0/20,GlobalMedia",
		"GEOIP,CN,Domestic",
		"MATCH,Final",
	}
}

// GenerateSingBoxSubscription 生成 sing-box 订阅
func (s *AccountService) GenerateSingBoxSubscription(accounts []*model.Account, serverIP string, reality *RealityConfig) (string, error) {
	outbounds := make([]map[string]interface{}, 0)
	for _, acc := range accounts {
		if !acc.Enabled {
			continue
		}
		var outbound map[string]interface{}
		if reality != nil && reality.Enabled {
			outbound = map[string]interface{}{
				"tag":         acc.Email,
				"type":       "vless",
				"server":     serverIP,
				"server_port": 443,
				"uuid":       acc.UUID,
				"flow":       "xtls-rprx-vision",
				"tls": map[string]interface{}{
					"enabled":    true,
					"server_name": reality.ServerName,
					"utls": map[string]interface{}{
						"enabled":    true,
						"fingerprint": "chrome",
					},
					"reality": map[string]interface{}{
						"enabled":   true,
						"public_key": reality.PublicKey,
						"short_id":  "6ba85179e30d4fc2",
					},
				},
			}
		} else {
			outbound = map[string]interface{}{
				"tag":         acc.Email,
				"type":       "vless",
				"server":     serverIP,
				"server_port": 443,
				"uuid":       acc.UUID,
				"flow":       "xtls-rprx-vision",
				"tls": map[string]interface{}{
					"enabled":    true,
					"server_name": serverIP,
				},
			}
		}
		outbounds = append(outbounds, outbound)
	}

	config := map[string]interface{}{
		"outbounds": outbounds,
	}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// SyncAllToRemote 同步服务器所有账号到远程
func (s *AccountService) SyncAllToRemote(serverID string, auth ssh.SSHAuth) error {
	server, err := s.serverRepo.GetByID(serverID)
	if err != nil {
		return err
	}
	if server == nil {
		return fmt.Errorf("server not found: %s", serverID)
	}

	accounts, err := s.accountRepo.ListByServerID(serverID)
	if err != nil {
		return err
	}

	client, err := ssh.NewSSHClient(server.IP, server.SSHPort, server.SSHUser, auth)
	if err != nil {
		return err
	}
	defer client.Close()

	sftpClient, err := ssh.NewSFTPClient(client)
	if err != nil {
		return err
	}
	defer sftpClient.Close()

	// Build clients array from all accounts
	clients := make([]map[string]interface{}, 0)
	for _, acc := range accounts {
		if acc.Enabled {
			clients = append(clients, map[string]interface{}{
				"id":    acc.UUID,
				"email": acc.Email,
			})
		}
	}

	// Generate full config
	config := map[string]interface{}{
		"log": map[string]interface{}{
			"loglevel": "warning",
		},
		"inbounds": []map[string]interface{}{
			{
				"port":     443,
				"protocol": "vless",
				"settings": map[string]interface{}{
					"clients": clients,
				},
			},
		},
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	tmpFile := "/tmp/v2ray_config_" + serverID + ".json"
	err = os.WriteFile(tmpFile, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write temp config: %w", err)
	}
	defer os.Remove(tmpFile)

	return sftpClient.UploadFile(tmpFile, "/etc/v2ray-agent/xray/conf/02_VLESS_TCP_inbounds.json")
}

// SyncToRemote 同步单个账号到远程服务器
func (s *AccountService) SyncToRemote(accountID string, auth ssh.SSHAuth) error {
	account, err := s.accountRepo.GetByID(accountID)
	if err != nil {
		return err
	}
	if account == nil {
		return fmt.Errorf("account not found: %s", accountID)
	}

	server, err := s.serverRepo.GetByID(account.ServerID)
	if err != nil {
		return err
	}
	if server == nil {
		return fmt.Errorf("server not found: %s", account.ServerID)
	}

	client, err := ssh.NewSSHClient(server.IP, server.SSHPort, server.SSHUser, auth)
	if err != nil {
		return err
	}
	defer client.Close()

	// Create SFTP client for file upload
	sftpClient, err := ssh.NewSFTPClient(client)
	if err != nil {
		return err
	}
	defer sftpClient.Close()

	// 生成配置文件内容
	config := map[string]interface{}{
		"log": map[string]interface{}{
			"loglevel": "warning",
		},
		"inbounds": []map[string]interface{}{
			{
				"port": 443,
				"protocol": "vless",
				"settings": map[string]interface{}{
					"clients": []map[string]interface{}{
						{
							"id": account.UUID,
							"email": account.Email,
						},
					},
				},
			},
		},
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write config to temp file and upload via SFTP
	tmpFile := "/tmp/v2ray_config_" + accountID + ".json"
	err = os.WriteFile(tmpFile, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to write temp config: %w", err)
	}
	defer os.Remove(tmpFile)

	return sftpClient.UploadFile(tmpFile, "/etc/v2ray-agent/xray/conf/02_VLESS_TCP_inbounds.json")
}

// ImportFromRemote 从远程服务器导入账号
func (s *AccountService) ImportFromRemote(serverID string, auth ssh.SSHAuth) ([]*model.Account, error) {
	server, err := s.serverRepo.GetByID(serverID)
	if err != nil {
		return nil, err
	}

	client, err := ssh.NewSSHClient(server.IP, server.SSHPort, server.SSHUser, auth)
	if err != nil {
		return nil, err
	}
	defer client.Close()

	// 尝试多个可能的配置文件路径
	configPaths := []string{
		"/etc/v2ray-agent/xray/conf/02_VLESS_TCP_inbounds.json",
		"/etc/v2ray-agent/xray/conf/07_VLESS_vision_reality_inbounds.json",
		"/etc/v2ray-agent/xray/conf/07_VLESS_reality_vision_inbounds.json",
	}

	var accounts []*model.Account
	var failed int
	var parsedPort int
	var publicKey string
	var serverName string
	var publicKeyFound bool

	// 获取现有账号列表（只查一次，避免N+1）
	existingAccounts, _ := s.accountRepo.ListByServerID(serverID)
	existingByUUID := make(map[string]*model.Account)
	for _, acc := range existingAccounts {
		existingByUUID[acc.UUID] = acc
	}

	for _, configPath := range configPaths {
		content, err := client.ReadRemoteFile(configPath)
		if err != nil {
			continue
		}

		// 尝试解析为标准格式
		var config struct {
			Inbounds []struct {
				Port     int `json:"port"`
				Listen   string `json:"listen"`
				Settings struct {
					Clients []struct {
						ID    string `json:"id"`
						Email string `json:"email"`
					} `json:"clients"`
				} `json:"settings"`
				StreamSettings struct {
					ServerName string   `json:"serverName"`
					Security   string   `json:"security"`
					RealitySettings struct {
						PublicKey   string   `json:"publicKey"`
						ServerNames []string `json:"serverNames"` // array, not string
						Target      string   `json:"target"`
					} `json:"realitySettings"`
				} `json:"streamSettings"`
			} `json:"inbounds"`
		}

		if err := json.Unmarshal([]byte(content), &config); err != nil {
			continue
		}

		for _, inbound := range config.Inbounds {
			// 记录端口（取第一个有效的）
			if parsedPort == 0 && inbound.Port > 0 {
				parsedPort = inbound.Port
			}
			// 记录 Reality publicKey 和 serverName
			rsPK := inbound.StreamSettings.RealitySettings.PublicKey
			rsTarget := inbound.StreamSettings.RealitySettings.Target
			ssSN := inbound.StreamSettings.ServerName
			// rsSN is now []string, get first element
			var rsSNStr string
			if len(inbound.StreamSettings.RealitySettings.ServerNames) > 0 {
				rsSNStr = inbound.StreamSettings.RealitySettings.ServerNames[0]
			}
			if rsPK != "" && !publicKeyFound {
				publicKey = rsPK
				// 优先从 realitySettings.serverNames 获取，其次从 streamSettings.serverName 获取，最后用 target
				if rsSNStr != "" {
					serverName = rsSNStr
				} else if ssSN != "" {
					serverName = ssSN
				} else if rsTarget != "" {
					// 从 target 提取 host:port 格式中的 host
					if idx := strings.Index(rsTarget, ":"); idx > 0 {
						serverName = rsTarget[:idx]
					} else {
						serverName = rsTarget
					}
				} else {
					serverName = server.IP
				}
				publicKeyFound = true
			}
			for _, client := range inbound.Settings.Clients {
				protocol := "vless_tcp"
				// 如果有 Reality 设置，则为 Reality 协议
				if rsPK != "" {
					protocol = "vless_reality_vision"
				}
				// 检查账号是否已存在（使用预加载的map）
				existingAccount := existingByUUID[client.ID]

				if existingAccount != nil {
					// 账号已存在，更新信息
					updateReq := &model.UpdateAccountRequest{
						Email:     &client.Email,
						Protocols: []string{protocol},
					}
					s.accountRepo.Update(existingAccount.ID, updateReq)
					// 获取更新后的账号
					updatedAccount, _ := s.accountRepo.GetByID(existingAccount.ID)
					if updatedAccount != nil {
						accounts = append(accounts, updatedAccount)
					}
					continue
				}

				account, err := s.accountRepo.Create(&model.CreateAccountRequest{
					ServerID:  serverID,
					UUID:      client.ID,
					Email:     client.Email,
					Protocols: []string{protocol},
				})
				if err != nil {
					failed++
					continue
				}
				accounts = append(accounts, account)
			}
		}
	}

	// 如果检测到 Reality 配置（publicKey 存在），更新服务器 Reality 配置
	realityEnabled := publicKeyFound

	// 如果解析到端口且检测到 Reality，更新服务器配置
	if parsedPort > 0 && publicKeyFound {
		s.serverRepo.Update(serverID, &model.UpdateServerRequest{
			RealityEnabled:   &realityEnabled,
			RealityPublicKey:  &publicKey,
			RealityServerName: &serverName,
			RealityPort:       &parsedPort,
		})
	}

	if failed > 0 && len(accounts) == 0 {
		return accounts, fmt.Errorf("failed to import %d account(s)", failed)
	}

	return accounts, nil
}

// GenerateVLESSSubscriptionMulti 生成多服务器 VLESS 订阅内容
func (s *AccountService) GenerateVLESSSubscriptionMulti(accounts []*model.Account, servers map[string]*model.Server, realityConfigs map[string]*RealityConfig) string {
	var lines []string
	for _, acc := range accounts {
		if !acc.Enabled {
			continue
		}
		server := servers[acc.ServerID]
		if server == nil {
			continue
		}
		reality := realityConfigs[acc.ServerID]
		link := s.GetAccountLink(acc, server.IP, "vless", reality)
		lines = append(lines, link)
	}
	return strings.Join(lines, "\n")
}

// GenerateClashMetaSubscriptionMulti 生成多服务器 Clash.Meta 订阅内容
func (s *AccountService) GenerateClashMetaSubscriptionMulti(accounts []*model.Account, servers map[string]*model.Server, realityConfigs map[string]*RealityConfig) (string, error) {
	proxies := make([]map[string]interface{}, 0)
	proxyNames := make([]string, 0)

	for _, acc := range accounts {
		if !acc.Enabled {
			continue
		}
		server := servers[acc.ServerID]
		if server == nil {
			continue
		}
		reality := realityConfigs[acc.ServerID]

		port := 443
		if reality != nil && reality.Enabled && reality.Port > 0 {
			port = reality.Port
		}

		nodeName := fmt.Sprintf("%s-%s", server.Name, acc.Email[:min(len(acc.Email), 8)])

		var proxy map[string]interface{}
		if reality != nil && reality.Enabled {
			proxy = map[string]interface{}{
				"name":                   nodeName,
				"type":                   "vless",
				"server":                 server.IP,
				"port":                   port,
				"uuid":                   acc.UUID,
				"network":                "tcp",
				"tls":                    true,
				"udp":                    true,
				"flow":                   "xtls-rprx-vision",
				"servername":             reality.ServerName,
				"client-fingerprint":     "chrome",
				"reality-opts": map[string]interface{}{
					"public-key": reality.PublicKey,
					"short-id":   "6ba85179e30d4fc2",
				},
			}
		} else {
			proxy = map[string]interface{}{
				"name":               nodeName,
				"type":               "vless",
				"server":             server.IP,
				"port":               port,
				"uuid":               acc.UUID,
				"network":            "tcp",
				"tls":                true,
				"udp":                true,
				"flow":               "xtls-rprx-vision",
				"client-fingerprint": "chrome",
			}
		}
		proxies = append(proxies, proxy)
		proxyNames = append(proxyNames, nodeName)
	}

	config := map[string]interface{}{
		"port":                     7890,
		"allow-lan":                true,
		"log-level":                "info",
		"external-controller":      "0.0.0.0:9090",
		"dns": map[string]interface{}{
			"enabled":          true,
			"listen":           "0.0.0.0:1053",
			"ipv6":              true,
			"enhanced-mode":    "fake-ip",
			"fake-ip-range":    "198.18.0.1/16",
			"fake-ip-filter": []string{
				"*.lan", "*.linksys.com", "*.linksyssmartwifi.com",
				"swscan.apple.com", "mesu.apple.com",
				"*.msftconnecttest.com", "*.msftncsi.com",
				"time.*.com", "time.*.gov", "time.*.edu.cn", "time.*.apple.com",
				"time1.*.com", "time2.*.com", "time3.*.com", "time4.*.com",
				"time5.*.com", "time6.*.com", "time7.*.com",
				"ntp.*.com", "ntp1.*.com", "ntp2.*.com", "ntp3.*.com", "ntp4.*.com",
				"*.time.edu.cn", "*.ntp.org.cn", "+.pool.ntp.org",
				"time1.cloud.tencent.com", "+.music.163.com", "*.126.net",
				"musicapi.taihe.com", "music.taihe.com",
				"songsearch.kugou.com", "trackercdn.kugou.com", "*.kuwo.cn",
				"api-jooxtt.sanook.com", "api.joox.com", "joox.com",
				"+.y.qq.com", "+.music.tc.qq.com", "aqqmusic.tc.qq.com",
				"+.stream.qqmusic.qq.com", "*.xiami.com", "+.music.migu.cn",
				"+.srv.nintendo.net", "+.stun.playstation.net",
				"xbox.*.microsoft.com", "+.xboxlive.com",
				"localhost.ptlogin2.qq.com", "proxy.golang.org", "stun.*.*",
				"stun.*.*.*", "*.mcdn.bilivideo.cn",
			},
			"default-nameserver": []string{"223.5.5.5", "114.114.114.114"},
			"nameserver": []string{
				"https://doh.pub/dns-query",
				"https://dns.alidns.com/dns-query",
			},
			"fallback-filter": map[string]interface{}{
				"geoip": false,
				"ipcidr": []string{"240.0.0.0/4", "0.0.0.0/32"},
			},
		},
		"proxies":       proxies,
		"proxy-groups": buildProxyGroups(proxyNames),
		"rules":        buildClashRules(),
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// GenerateSingBoxSubscriptionMulti 生成多服务器 Sing-box 订阅内容
func (s *AccountService) GenerateSingBoxSubscriptionMulti(accounts []*model.Account, servers map[string]*model.Server, realityConfigs map[string]*RealityConfig) (string, error) {
	outbounds := make([]map[string]interface{}, 0)

	for _, acc := range accounts {
		if !acc.Enabled {
			continue
		}
		server := servers[acc.ServerID]
		if server == nil {
			continue
		}
		reality := realityConfigs[acc.ServerID]

		var outbound map[string]interface{}
		if reality != nil && reality.Enabled {
			outbound = map[string]interface{}{
				"tag":         fmt.Sprintf("%s-%s", server.Name, acc.Email[:8]),
				"type":       "vless",
				"server":     server.IP,
				"server_port": 443,
				"uuid":       acc.UUID,
				"flow":       "xtls-rprx-vision",
				"tls": map[string]interface{}{
					"enabled":    true,
					"server_name": reality.ServerName,
					"utls": map[string]interface{}{
						"enabled":    true,
						"fingerprint": "chrome",
					},
					"reality": map[string]interface{}{
						"enabled":   true,
						"public_key": reality.PublicKey,
						"short_id":  "6ba85179e30d4fc2",
					},
				},
			}
		} else {
			outbound = map[string]interface{}{
				"tag":         fmt.Sprintf("%s-%s", server.Name, acc.Email[:8]),
				"type":       "vless",
				"server":     server.IP,
				"server_port": 443,
				"uuid":       acc.UUID,
				"flow":       "xtls-rprx-vision",
				"tls": map[string]interface{}{
					"enabled":    true,
					"server_name": server.IP,
				},
			}
		}
		outbounds = append(outbounds, outbound)
	}

	config := map[string]interface{}{
		"outbounds": outbounds,
	}
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}