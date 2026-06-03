<!DOCTYPE html>
<html lang="zh-CN">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>监控中心</title>
  <link rel="preconnect" href="https://fonts.googleapis.com">
  <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
  <link href="https://fonts.googleapis.com/css2?family=JetBrains+Mono:wght@400;500;600;700&family=Syne:wght@400;500;600;700;800&display=swap" rel="stylesheet">
  <script src="https://cdn.plot.ly/plotly-2.27.0.min.js"></script>
  <style>
    :root {
      --bg-deep: #050508;
      --bg-card: #0c0c14;
      --bg-card-hover: #12121e;
      --border-subtle: rgba(0, 255, 242, 0.1);
      --border-glow: rgba(0, 255, 242, 0.4);
      --cyan: #00fff2;
      --cyan-dim: rgba(0, 255, 242, 0.15);
      --magenta: #ff00aa;
      --magenta-dim: rgba(255, 0, 170, 0.15);
      --green: #00ff88;
      --green-dim: rgba(0, 255, 136, 0.15);
      --yellow: #ffaa00;
      --red: #ff4466;
      --text-primary: #e0e0e8;
      --text-secondary: #6a6a7a;
      --text-dim: #3a3a4a;
    }

    * {
      margin: 0;
      padding: 0;
      box-sizing: border-box;
    }

    body {
      font-family: 'Syne', sans-serif;
      background: var(--bg-deep);
      color: var(--text-primary);
      min-height: 100vh;
      overflow-x: hidden;
    }

    /* Animated background grid */
    .bg-grid {
      position: fixed;
      top: 0;
      left: 0;
      right: 0;
      bottom: 0;
      background-image:
        linear-gradient(rgba(0, 255, 242, 0.03) 1px, transparent 1px),
        linear-gradient(90deg, rgba(0, 255, 242, 0.03) 1px, transparent 1px);
      background-size: 50px 50px;
      pointer-events: none;
      z-index: 0;
    }

    .monitor-container {
      position: relative;
      z-index: 1;
      padding: 32px 40px;
      max-width: 1600px;
      margin: 0 auto;
    }

    /* Header */
    .monitor-header {
      display: flex;
      justify-content: space-between;
      align-items: flex-start;
      margin-bottom: 40px;
      padding-bottom: 24px;
      border-bottom: 1px solid var(--border-subtle);
    }

    .header-left h1 {
      font-family: 'Syne', sans-serif;
      font-size: 32px;
      font-weight: 800;
      color: var(--text-primary);
      letter-spacing: -0.5px;
      margin-bottom: 8px;
    }

    .header-left p {
      font-size: 14px;
      color: var(--text-secondary);
      font-family: 'JetBrains Mono', monospace;
    }

    .time-tabs {
      display: flex;
      gap: 8px;
      background: var(--bg-card);
      padding: 6px;
      border-radius: 12px;
      border: 1px solid var(--border-subtle);
    }

    .time-tab {
      padding: 10px 20px;
      background: transparent;
      border: none;
      color: var(--text-secondary);
      font-family: 'JetBrains Mono', monospace;
      font-size: 13px;
      cursor: pointer;
      border-radius: 8px;
      transition: all 0.3s ease;
    }

    .time-tab:hover {
      color: var(--cyan);
      background: var(--cyan-dim);
    }

    .time-tab.active {
      background: linear-gradient(135deg, var(--cyan-dim), var(--magenta-dim));
      color: var(--cyan);
      box-shadow: 0 0 20px var(--cyan-dim);
    }

    /* Server Grid */
    .server-grid {
      display: grid;
      grid-template-columns: 280px 1fr;
      gap: 24px;
    }

    /* Server Card (Left Panel) */
    .server-card {
      background: var(--bg-card);
      border: 1px solid var(--border-subtle);
      border-radius: 16px;
      padding: 24px;
      position: relative;
      overflow: hidden;
      transition: all 0.4s ease;
    }

    .server-card::before {
      content: '';
      position: absolute;
      top: 0;
      left: 0;
      right: 0;
      height: 3px;
      background: linear-gradient(90deg, var(--cyan), var(--magenta));
      opacity: 0;
      transition: opacity 0.3s ease;
    }

    .server-card:hover::before {
      opacity: 1;
    }

    .server-card:hover {
      border-color: var(--border-glow);
      box-shadow: 0 0 40px rgba(0, 255, 242, 0.1);
    }

    .server-status-indicator {
      display: flex;
      align-items: center;
      gap: 12px;
      margin-bottom: 20px;
    }

    .status-dot {
      width: 12px;
      height: 12px;
      border-radius: 50%;
      position: relative;
    }

    .status-dot.online {
      background: var(--green);
      box-shadow: 0 0 12px var(--green);
      animation: pulse-green 2s ease-in-out infinite;
    }

    .status-dot.offline {
      background: var(--red);
      box-shadow: 0 0 12px var(--red);
    }

    @keyframes pulse-green {
      0%, 100% { box-shadow: 0 0 12px var(--green); }
      50% { box-shadow: 0 0 24px var(--green), 0 0 48px var(--green-dim); }
    }

    .status-label {
      font-family: 'JetBrains Mono', monospace;
      font-size: 12px;
      text-transform: uppercase;
      letter-spacing: 1px;
    }

    .status-label.online { color: var(--green); }
    .status-label.offline { color: var(--red); }

    .server-name {
      font-size: 24px;
      font-weight: 700;
      margin-bottom: 4px;
      color: var(--text-primary);
    }

    .server-ip {
      font-family: 'JetBrains Mono', monospace;
      font-size: 14px;
      color: var(--cyan);
      margin-bottom: 24px;
    }

    /* Stats Grid */
    .stats-grid {
      display: grid;
      grid-template-columns: 1fr 1fr;
      gap: 16px;
    }

    .stat-item {
      background: rgba(0, 0, 0, 0.3);
      padding: 16px;
      border-radius: 12px;
      border: 1px solid var(--border-subtle);
    }

    .stat-label {
      font-size: 11px;
      color: var(--text-secondary);
      text-transform: uppercase;
      letter-spacing: 1px;
      margin-bottom: 8px;
      font-family: 'JetBrains Mono', monospace;
    }

    .stat-value {
      font-size: 20px;
      font-weight: 600;
      font-family: 'JetBrains Mono', monospace;
    }

    .stat-value.cpu { color: var(--cyan); }
    .stat-value.memory { color: var(--magenta); }
    .stat-value.disk { color: var(--yellow); }
    .stat-value.bandwidth { color: var(--green); }

    .stat-unit {
      font-size: 12px;
      color: var(--text-secondary);
      margin-left: 4px;
    }

    /* Metrics Panel (Right) */
    .metrics-panel {
      display: flex;
      flex-direction: column;
      gap: 20px;
    }

    .metric-card {
      background: var(--bg-card);
      border: 1px solid var(--border-subtle);
      border-radius: 16px;
      padding: 24px;
      position: relative;
      overflow: hidden;
    }

    .metric-card:hover {
      border-color: var(--border-glow);
    }

    .metric-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 20px;
    }

    .metric-title {
      font-size: 14px;
      font-weight: 600;
      color: var(--text-primary);
      display: flex;
      align-items: center;
      gap: 10px;
    }

    .metric-indicator {
      width: 8px;
      height: 8px;
      border-radius: 50%;
    }

    .metric-indicator.cpu { background: var(--cyan); box-shadow: 0 0 8px var(--cyan); }
    .metric-indicator.memory { background: var(--magenta); box-shadow: 0 0 8px var(--magenta); }
    .metric-indicator.bandwidth { background: var(--green); box-shadow: 0 0 8px var(--green); }

    .metric-value-current {
      font-family: 'JetBrains Mono', monospace;
      font-size: 24px;
      font-weight: 600;
    }

    .metric-value-current.highlight {
      color: var(--cyan);
    }

    .chart-container {
      height: 120px;
      position: relative;
    }

    /* Bandwidth specific */
    .bandwidth-chart {
      display: grid;
      grid-template-columns: 1fr 1fr;
      gap: 16px;
    }

    .bandwidth-item {
      position: relative;
    }

    .bandwidth-label {
      display: flex;
      align-items: center;
      gap: 8px;
      margin-bottom: 12px;
    }

    .bandwidth-arrow {
      font-size: 16px;
    }

    .bandwidth-arrow.in { color: var(--cyan); }
    .bandwidth-arrow.out { color: var(--magenta); }

    .bandwidth-title {
      font-size: 12px;
      color: var(--text-secondary);
      font-family: 'JetBrains Mono', monospace;
      text-transform: uppercase;
      letter-spacing: 1px;
    }

    .bandwidth-chart-container {
      height: 100px;
    }

    /* Empty state */
    .empty-state {
      grid-column: 1 / -1;
      text-align: center;
      padding: 80px 40px;
      background: var(--bg-card);
      border: 1px dashed var(--border-subtle);
      border-radius: 16px;
    }

    .empty-state h3 {
      font-size: 20px;
      margin-bottom: 8px;
      color: var(--text-primary);
    }

    .empty-state p {
      color: var(--text-secondary);
      font-size: 14px;
    }

    /* Animations */
    @keyframes fadeInUp {
      from {
        opacity: 0;
        transform: translateY(20px);
      }
      to {
        opacity: 1;
        transform: translateY(0);
      }
    }

    .animate-in {
      animation: fadeInUp 0.6s ease-out;
    }

    .animate-delay-1 { animation-delay: 0.1s; }
    .animate-delay-2 { animation-delay: 0.2s; }
    .animate-delay-3 { animation-delay: 0.3s; }

    /* Responsive */
    @media (max-width: 1200px) {
      .server-grid {
        grid-template-columns: 1fr;
      }
    }

    @media (max-width: 768px) {
      .monitor-container {
        padding: 20px;
      }
      .monitor-header {
        flex-direction: column;
        gap: 20px;
      }
      .header-left h1 {
        font-size: 24px;
      }
      .bandwidth-chart {
        grid-template-columns: 1fr;
      }
    }
  </style>
</head>
<body>
  <div class="bg-grid"></div>

  <div class="monitor-container animate-in">
    <header class="monitor-header">
      <div class="header-left">
        <h1>监控中心</h1>
        <p>// REAL-TIME SERVER TELEMETRY</p>
      </div>
      <div class="time-tabs">
        <button class="time-tab active" data-range="1h">1H</button>
        <button class="time-tab" data-range="4h">4H</button>
        <button class="time-tab" data-range="12h">12H</button>
        <button class="time-tab" data-range="24h">24H</button>
      </div>
    </header>

    <div id="server-list"></div>
  </div>

  <script>
    const TIME_RANGES = { '1h': '1小时', '4h': '4小时', '12h': '12小时', '24h': '24小时' };
    let currentTimeRange = '1h';
    let servers = [];
    let statuses = new Map();
    let chartInstances = {};

    // Load data
    async function loadData() {
      try {
        const [serverData, statusData] = await Promise.all([
          fetch('/api/servers').then(r => r.json()),
          fetch(`/api/logs/node-status?time_range=${currentTimeRange}`).then(r => r.json())
        ]);

        servers = serverData || [];
        statuses = new Map();
        if (statusData && statusData.length > 0) {
          statusData.forEach(s => statuses.set(s.server_id, s));
        }

        renderServers();
      } catch (e) {
        console.error('Load failed:', e);
      }
    }

    // Format bytes
    function formatBytes(bytes) {
      if (bytes === 0) return '0 B';
      const k = 1024;
      const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
      const i = Math.floor(Math.log(bytes) / Math.log(k));
      return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
    }

    // Render servers
    function renderServers() {
      const container = document.getElementById('server-list');

      if (servers.length === 0) {
        container.innerHTML = `
          <div class="empty-state">
            <h3>暂无服务器</h3>
            <p>请先在服务器管理中添加服务器</p>
          </div>
        `;
        return;
      }

      container.innerHTML = servers.map((server, index) => {
        const status = statuses.get(server.id);
        const isOnline = status && status.current && status.current.v2ray_status === 'running';
        const cpu = status?.current?.cpu_percent || 0;
        const memory = status?.current?.memory_percent || 0;
        const disk = status?.current?.disk_percent || 0;
        const bandwidthIn = status?.current?.bandwidth_in || 0;
        const bandwidthOut = status?.current?.bandwidth_out || 0;

        return `
          <div class="server-grid animate-in animate-delay-${index + 1}">
            <div class="server-card">
              <div class="server-status-indicator">
                <div class="status-dot ${isOnline ? 'online' : 'offline'}"></div>
                <span class="status-label ${isOnline ? 'online' : 'offline'}">${isOnline ? 'ONLINE' : 'OFFLINE'}</span>
              </div>
              <div class="server-name">${server.name}</div>
              <div class="server-ip">${server.ip}</div>

              <div class="stats-grid">
                <div class="stat-item">
                  <div class="stat-label">CPU</div>
                  <div class="stat-value cpu">${cpu.toFixed(1)}<span class="stat-unit">%</span></div>
                </div>
                <div class="stat-item">
                  <div class="stat-label">内存</div>
                  <div class="stat-value memory">${memory.toFixed(1)}<span class="stat-unit">%</span></div>
                </div>
                <div class="stat-item">
                  <div class="stat-label">磁盘</div>
                  <div class="stat-value disk">${disk.toFixed(0)}<span class="stat-unit">%</span></div>
                </div>
                <div class="stat-item">
                  <div class="stat-label">带宽</div>
                  <div class="stat-value bandwidth">${formatBytes(bandwidthIn + bandwidthOut)}<span class="stat-unit">/s</span></div>
                </div>
              </div>
            </div>

            <div class="metrics-panel">
              ${status ? `
                <div class="metric-card">
                  <div class="metric-header">
                    <div class="metric-title">
                      <div class="metric-indicator cpu"></div>
                      CPU 使用率
                    </div>
                    <div class="metric-value-current highlight">${cpu.toFixed(1)}%</div>
                  </div>
                  <div class="chart-container" id="cpu-chart-${server.id}"></div>
                </div>

                <div class="metric-card">
                  <div class="metric-header">
                    <div class="metric-title">
                      <div class="metric-indicator memory"></div>
                      内存使用率
                    </div>
                    <div class="metric-value-current" style="color: var(--magenta)">${memory.toFixed(1)}%</div>
                  </div>
                  <div class="chart-container" id="memory-chart-${server.id}"></div>
                </div>

                <div class="metric-card">
                  <div class="metric-header">
                    <div class="metric-title">
                      <div class="metric-indicator bandwidth"></div>
                      带宽流量
                    </div>
                  </div>
                  <div class="bandwidth-chart">
                    <div class="bandwidth-item">
                      <div class="bandwidth-label">
                        <span class="bandwidth-arrow in">↓</span>
                        <span class="bandwidth-title">入站</span>
                      </div>
                      <div class="bandwidth-chart-container" id="bandwidth-in-chart-${server.id}"></div>
                    </div>
                    <div class="bandwidth-item">
                      <div class="bandwidth-label">
                        <span class="bandwidth-arrow out">↑</span>
                        <span class="bandwidth-title">出站</span>
                      </div>
                      <div class="bandwidth-chart-container" id="bandwidth-out-chart-${server.id}"></div>
                    </div>
                  </div>
                </div>
              ` : `
                <div class="metric-card" style="display: flex; align-items: center; justify-content: center; min-height: 300px;">
                  <div style="text-align: center; color: var(--text-secondary);">
                    <div style="font-size: 48px; margin-bottom: 16px;">⏳</div>
                    <div>等待 Agent 上报状态...</div>
                  </div>
                </div>
              `}
            </div>
          </div>
        `;
      }).join('');

      // Render charts after DOM is updated
      if (status) {
        servers.forEach(server => {
          const serverStatus = statuses.get(server.id);
          if (serverStatus) {
            renderCharts(server.id, serverStatus);
          }
        });
      }
    }

    // Render charts
    function renderCharts(serverId, status) {
      const plotConfig = {
        responsive: true,
        displayModeBar: false,
        legend: { display: false },
      };

      const layout = {
        margin: { t: 10, r: 10, b: 30, l: 40 },
        paper_bgcolor: 'transparent',
        plot_bgcolor: 'transparent',
        font: { color: '#6a6a7a', family: 'JetBrains Mono' },
        xaxis: {
          showgrid: true,
          gridcolor: 'rgba(0, 255, 242, 0.05)',
          linecolor: 'rgba(0, 255, 242, 0.1)',
          tickcolor: 'rgba(0, 255, 242, 0.1)',
          ticks: 'outside',
          tickfont: { size: 10 },
        },
        yaxis: {
          showgrid: true,
          gridcolor: 'rgba(0, 255, 242, 0.05)',
          linecolor: 'rgba(0, 255, 242, 0.1)',
          tickcolor: 'rgba(0, 255, 242, 0.1)',
          ticks: 'outside',
          tickfont: { size: 10 },
          rangemode: 'tozero',
        },
      };

      // CPU Chart
      const cpuData = [{
        x: status.metrics.cpu.map(p => p.time),
        y: status.metrics.cpu.map(p => p.value),
        type: 'scatter',
        mode: 'lines',
        line: { color: '#00fff2', width: 2, shape: 'spline' },
        fill: 'tozeroy',
        fillcolor: 'rgba(0, 255, 242, 0.1)',
      }];
      Plotly.newPlot(`cpu-chart-${serverId}`, cpuData, { ...layout, yaxis: { ...layout.yaxis, title: '%' } }, plotConfig);

      // Memory Chart
      const memoryData = [{
        x: status.metrics.memory.map(p => p.time),
        y: status.metrics.memory.map(p => p.value),
        type: 'scatter',
        mode: 'lines',
        line: { color: '#ff00aa', width: 2, shape: 'spline' },
        fill: 'tozeroy',
        fillcolor: 'rgba(255, 0, 170, 0.1)',
      }];
      Plotly.newPlot(`memory-chart-${serverId}`, memoryData, { ...layout, yaxis: { ...layout.yaxis, title: '%' } }, plotConfig);

      // Bandwidth In Chart
      const bandwidthInData = [{
        x: status.metrics.bandwidth_in.map(p => p.time),
        y: status.metrics.bandwidth_in.map(p => p.value),
        type: 'scatter',
        mode: 'lines',
        line: { color: '#00fff2', width: 2, shape: 'spline' },
        fill: 'tozeroy',
        fillcolor: 'rgba(0, 255, 242, 0.15)',
      }];
      Plotly.newPlot(`bandwidth-in-chart-${serverId}`, bandwidthInData, { ...layout, yaxis: { ...layout.yaxis, title: 'B' } }, plotConfig);

      // Bandwidth Out Chart
      const bandwidthOutData = [{
        x: status.metrics.bandwidth_out.map(p => p.time),
        y: status.metrics.bandwidth_out.map(p => p.value),
        type: 'scatter',
        mode: 'lines',
        line: { color: '#ff00aa', width: 2, shape: 'spline' },
        fill: 'tozeroy',
        fillcolor: 'rgba(255, 0, 170, 0.15)',
      }];
      Plotly.newPlot(`bandwidth-out-chart-${serverId}`, bandwidthOutData, { ...layout, yaxis: { ...layout.yaxis, title: 'B' } }, plotConfig);
    }

    // Time tab click handler
    document.addEventListener('click', (e) => {
      if (e.target.classList.contains('time-tab')) {
        document.querySelectorAll('.time-tab').forEach(tab => tab.classList.remove('active'));
        e.target.classList.add('active');
        currentTimeRange = e.target.dataset.range;
        loadData();
      }
    });

    // Initial load
    loadData();

    // Auto refresh every 30 seconds
    setInterval(loadData, 30000);
  </script>
</body>
</html>