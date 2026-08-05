package main

const adminHTML = `<!DOCTYPE html>
<html lang="zh-CN" data-theme="light">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>AI Models Gateway</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
/* 白色主题（默认） */
:root{
--bg:#f5f7fa;--card:#ffffff;--border:#e2e8f0;--accent:#3b82f6;--text:#1e293b;
--muted:#64748b;--green:#10b981;--blue:#3b82f6;--red:#ef4444;--radius:8px;
--shadow:0 1px 3px rgba(0,0,0,.08);--hover:#f1f5f9;--input-bg:#fff;--tag-bg:#e0e7ff;--tag-text:#3730a3
}
[data-theme="dark"]{
--bg:#1a1a2e;--card:#16213e;--border:#0f3460;--accent:#e94560;--text:#eee;
--muted:#999;--green:#4ecca3;--blue:#4fc3f7;--red:#e94560;--radius:8px;
--shadow:0 1px 3px rgba(0,0,0,.3);--hover:rgba(255,255,255,.04);--input-bg:#1a1a2e;--tag-bg:rgba(79,195,247,.15);--tag-text:#4fc3f7
}
body{font-family:'Segoe UI',system-ui,sans-serif;background:var(--bg);color:var(--text);min-height:100vh;transition:background .2s,color .2s}
.header{background:var(--card);border-bottom:2px solid var(--accent);padding:16px 24px;display:flex;align-items:center;justify-content:space-between;box-shadow:var(--shadow)}
.header-left{display:flex;align-items:center;gap:16px}
.header h1{font-size:20px;font-weight:600}
.header h1 span{color:var(--accent)}
.header .info{font-size:13px;color:var(--muted)}
.header-right{display:flex;align-items:center;gap:12px}
.theme-toggle{width:38px;height:38px;border-radius:50%;border:1px solid var(--border);background:var(--card);cursor:pointer;font-size:16px;display:flex;align-items:center;justify-content:center;transition:.2s}
.theme-toggle:hover{border-color:var(--accent);transform:scale(1.05)}
.container{max-width:1400px;margin:0 auto;padding:24px}
.tabs{display:flex;gap:4px;margin-bottom:20px}
.tab{padding:10px 20px;background:var(--card);border:1px solid var(--border);border-radius:var(--radius) var(--radius) 0 0;cursor:pointer;color:var(--muted);font-size:14px;transition:.2s}
.tab:hover{color:var(--text);background:var(--hover)}
.tab.active{background:var(--accent);color:#fff;border-color:var(--accent)}
.panel{display:none}
.panel.active{display:block}
.card{background:var(--card);border:1px solid var(--border);border-radius:var(--radius);padding:20px;margin-bottom:16px;box-shadow:var(--shadow)}
.card-title{font-size:16px;font-weight:600;margin-bottom:16px;display:flex;justify-content:space-between;align-items:center}
table{width:100%;border-collapse:collapse;font-size:14px}
th{text-align:left;padding:10px 12px;border-bottom:2px solid var(--border);color:var(--muted);font-weight:500;font-size:12px;text-transform:uppercase}
td{padding:10px 12px;border-bottom:1px solid var(--border);vertical-align:middle}
tr:hover{background:var(--hover)}
.badge{display:inline-block;padding:2px 8px;border-radius:4px;font-size:12px;font-weight:500}
.badge-active{background:rgba(16,185,129,.15);color:var(--green)}
.badge-disabled{background:rgba(239,68,68,.15);color:var(--red)}
.badge-openai{background:rgba(59,130,246,.15);color:var(--blue)}
.badge-anthropic{background:rgba(239,68,68,.15);color:var(--red)}
.badge-current{background:var(--accent);color:#fff}
.btn{padding:8px 16px;border:none;border-radius:var(--radius);cursor:pointer;font-size:13px;font-weight:500;transition:.2s}
.btn-primary{background:var(--accent);color:#fff}
.btn-primary:hover{opacity:.88}
.btn-sm{padding:4px 10px;font-size:12px}
.btn-danger{background:rgba(239,68,68,.12);color:var(--red);border:1px solid rgba(239,68,68,.25)}
.btn-danger:hover{background:rgba(239,68,68,.22)}
.btn-success{background:rgba(16,185,129,.12);color:var(--green);border:1px solid rgba(16,185,129,.25)}
.btn-success:hover{background:rgba(16,185,129,.22)}
.btn-outline{background:transparent;border:1px solid var(--border);color:var(--text)}
.btn-outline:hover{border-color:var(--accent)}
.input,.select,textarea{width:100%;padding:8px 12px;background:var(--input-bg);border:1px solid var(--border);border-radius:var(--radius);color:var(--text);font-size:14px;font-family:inherit}
.input:focus,.select:focus,textarea:focus{outline:none;border-color:var(--accent)}
.form-group{margin-bottom:14px}
.form-group label{display:block;margin-bottom:6px;font-size:13px;color:var(--muted)}
.form-row{display:flex;gap:12px}
.form-row .form-group{flex:1}
.modal-overlay{position:fixed;inset:0;background:rgba(0,0,0,.4);display:none;align-items:center;justify-content:center;z-index:100}
.modal-overlay.show{display:flex}
.modal{background:var(--card);border:1px solid var(--border);border-radius:var(--radius);padding:24px;width:640px;max-width:95vw;max-height:88vh;overflow-y:auto;box-shadow:0 10px 40px rgba(0,0,0,.2)}
.modal-title{font-size:18px;font-weight:600;margin-bottom:20px}
.modal-actions{display:flex;gap:8px;justify-content:flex-end;margin-top:20px}
.stats-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(200px,1fr));gap:16px;margin-bottom:20px}
.stat-card{background:var(--card);border:1px solid var(--border);border-radius:var(--radius);padding:20px;text-align:center;box-shadow:var(--shadow)}
.stat-value{font-size:28px;font-weight:700;color:var(--accent)}
.stat-label{font-size:13px;color:var(--muted);margin-top:4px}
.toast{position:fixed;top:80px;right:24px;padding:12px 20px;border-radius:var(--radius);font-size:14px;z-index:200;animation:slideInRight .3s;max-width:400px;box-shadow:0 4px 12px rgba(0,0,0,.15)}
.toast-success{background:var(--green);color:#fff}
.toast-error{background:var(--red);color:#fff}
@keyframes slideInRight{from{transform:translateX(120%);opacity:0}to{transform:translateX(0);opacity:1}}
.empty{text-align:center;padding:40px;color:var(--muted)}
.mono{font-family:'Courier New',monospace;font-size:13px}
.test-result{margin-top:12px;padding:12px;border-radius:var(--radius);font-size:13px;max-width:100%;overflow-x:hidden}
.test-success{background:rgba(16,185,129,.08);border:1px solid rgba(16,185,129,.25);word-break:break-word;overflow-wrap:anywhere}
.test-error{background:rgba(239,68,68,.08);border:1px solid rgba(239,68,68,.25);word-break:break-word;overflow-wrap:anywhere}
.loading{display:inline-block;width:14px;height:14px;border:2px solid var(--border);border-top-color:var(--accent);border-radius:50%;animation:spin .6s linear infinite}
@keyframes spin{to{transform:rotate(360deg)}}
.config-box{background:var(--bg);border:1px solid var(--border);border-radius:var(--radius);padding:16px;margin-top:16px}
.config-box h4{font-size:13px;color:var(--muted);margin-bottom:8px}
.config-line{font-family:'Courier New',monospace;font-size:13px;padding:4px 0;color:var(--green)}
/* 模型标签 */
.tag-input-wrap{border:1px solid var(--border);border-radius:var(--radius);padding:6px 8px;background:var(--input-bg);display:flex;flex-wrap:wrap;gap:6px;align-items:center;min-height:38px}
.tag-input-wrap:focus-within{border-color:var(--accent)}
.tag{display:inline-flex;align-items:center;gap:4px;background:var(--tag-bg);color:var(--tag-text);padding:3px 8px;border-radius:4px;font-size:13px;font-family:'Courier New',monospace}
.tag .tag-x{cursor:pointer;font-weight:700;opacity:.6;border:none;background:none;color:inherit;font-size:14px;padding:0;line-height:1}
.tag .tag-x:hover{opacity:1}
.tag-input{border:none;outline:none;background:transparent;color:var(--text);font-size:14px;font-family:inherit;flex:1;min-width:120px;padding:4px}
/* 模型列表中的标签 */
.model-chip{display:flex;align-items:center;gap:6px;background:var(--tag-bg);color:var(--tag-text);padding:4px 8px;border-radius:4px;font-size:12px;font-family:'Courier New',monospace;margin:4px 0;border:1px solid var(--border)}
.model-chip.disabled{opacity:.45}
.model-chip .chip-toggle{cursor:pointer;border:none;background:none;color:inherit;font-size:12px;padding:0;flex-shrink:0}
#statsByProvider table,#statsByModel table{font-size:12px}
#statsByProvider,#statsByModel{max-height:200px;overflow-y:auto}
.model-chip .chip-toggle:hover{opacity:.8}
.model-chip .chip-name{flex:1;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.model-chip.disabled .chip-name{text-decoration:line-through}
.model-chip .chip-actions{display:flex;align-items:center;gap:4px;flex-wrap:wrap;flex-shrink:1}
.model-row{margin-top:8px}
.url-hint{font-size:11px;color:var(--muted);font-family:'Courier New',monospace;margin-top:4px;word-break:break-all}
.expand-btn{font-size:12px;color:var(--accent);cursor:pointer;background:none;border:none;padding:2px 6px}
.copy-btn{display:inline-flex;align-items:center;gap:4px;padding:2px 8px;font-size:11px;border-radius:4px;border:1px solid var(--border);background:var(--card);color:var(--muted);cursor:pointer;transition:.2s;vertical-align:middle}
.copy-btn:hover{border-color:var(--accent);color:var(--accent)}
.copy-btn.copied{background:var(--green);color:#fff;border-color:var(--green)}
.url-with-copy{display:flex;align-items:center;gap:6px;flex-wrap:wrap}
.chat-msg{max-width:80%;padding:8px 12px;border-radius:8px;margin-bottom:8px;word-break:break-word;white-space:pre-wrap;font-size:14px;line-height:1.5}
.chat-msg.user{background:var(--accent);color:#fff;margin-left:auto}
.chat-msg.assistant{background:var(--card);border:1px solid var(--border);margin-right:auto}
.chat-msg.error{background:#fee;color:#c33;border:1px solid #fcc;margin-right:auto}
.chat-msg .role{font-size:11px;opacity:.7;margin-bottom:2px}
.key-item{display:flex;align-items:center;gap:8px;padding:6px 8px;background:var(--bg);border:1px solid var(--border);border-radius:var(--radius);margin-bottom:4px;font-size:13px}
.key-item .key-val{font-family:'Courier New',monospace;flex:1;overflow:hidden;text-overflow:ellipsis;white-space:nowrap}
.key-item .key-name{color:var(--muted);font-size:12px}
.key-item .key-x{cursor:pointer;border:none;background:none;color:var(--red);font-size:14px;padding:0 4px}
</style>
</head>
<body>
<div class="header">
<div class="header-left">
<h1>AI Models <span>Gateway</span></h1>
<div style="display:flex;align-items:center;gap:12px">
<div class="info" id="headerInfo">加载中...</div>
<span style="font-size:11px;color:var(--muted);opacity:.6">v{{VERSION}}</span>
</div>
</div>
<div class="header-right">
<button class="theme-toggle" onclick="toggleTheme()" id="themeBtn" title="切换主题">🌙</button>
</div>
</div>
<div class="container">
<div class="tabs">
<button class="tab active" onclick="switchTab(event,'providers')">中转站管理</button>
<button class="tab" onclick="switchTab(event,'aliases')">模型路由</button>
<button class="tab" onclick="switchTab(event,'keys')">API Keys</button>
<button class="tab" onclick="switchTab(event,'stats')">用量统计</button>
<button class="tab" onclick="switchTab(event,'chat')">聊天测试</button>
<button class="tab" onclick="switchTab(event,'settings')">设置</button>
<button class="tab" onclick="switchTab(event,'config')">接入配置</button>
</div>

<!-- 中转站管理 -->
<div class="panel active" id="panel-providers">
<div class="card">
<div class="card-title">中转站列表 <div style="display:flex;gap:8px"><button class="btn btn-outline" onclick="importProvider()">导入配置</button><button class="btn btn-outline" onclick="showOpenCodeModal()">导入OpenCode</button><button class="btn btn-primary" onclick="showProviderModal()">+ 添加中转站</button></div></div>
<div style="display:flex;gap:8px;align-items:center;margin-bottom:12px">
<label style="font-size:13px;color:var(--muted);display:flex;align-items:center;gap:4px;cursor:pointer"><input type="checkbox" id="provSelectAll" onchange="toggleAllProviders(this.checked)"> 全选</label>
<span class="badge badge-active" id="provSelCount">已选 0</span>
<button class="btn btn-sm btn-outline" onclick="testAllProviders()">⚡ 一键检测</button>
<button class="btn btn-sm btn-outline" onclick="batchDeleteProviders()">🗑 批量删除</button>
</div>
<div id="providerList"></div>
<input type="file" id="importFile" accept=".json" style="display:none" onchange="handleImportFile(event)">
</div>
</div>

<!-- 模型路由 -->
<div class="panel" id="panel-aliases">
<div class="card">
<div class="card-title">全局默认路由</div>
<p style="color:var(--muted);font-size:13px;margin-bottom:16px">客户端不指定站点时，默认路由到当前启用的站点和默认模型。</p>
<div id="globalRouteInfo"></div>
</div>
<div class="card">
<div class="card-title">模型路由别名 <button class="btn btn-primary" onclick="showAliasModal()">+ 添加别名</button></div>
<p style="color:var(--muted);font-size:13px;margin-bottom:16px">客户端用固定的模型名调用网关，网关自动路由到指定的站点和模型。切换实际模型时只需修改别名，无需改客户端配置。</p>
<div id="aliasList"></div>
</div>
</div>

<!-- API Keys -->
<div class="panel" id="panel-keys">
<div class="card">
<div class="card-title">网关 API Keys <button class="btn btn-primary" onclick="showKeyModal()">+ 生成 Key</button></div>
<p style="color:var(--muted);font-size:13px;margin-bottom:16px">客户端调用网关时使用的密钥。未配置 Key 时允许无鉴权访问。</p>
<div id="keyList"></div>
</div>
</div>

<!-- 统计 -->
<div class="panel" id="panel-stats">
<div class="stats-grid" id="statsGrid"></div>
<div class="card">
<div class="card-title">用量趋势（按日期）</div>
<div style="position:relative;height:280px">
<canvas id="chartDaily"></canvas>
</div>
</div>
<div class="card">
<div class="card-title">按中转站统计</div>
<div style="position:relative;height:280px">
<canvas id="chartProvider"></canvas>
</div>
<div id="statsByProvider" style="margin-top:12px"></div>
</div>
<div class="card">
<div class="card-title">按模型统计</div>
<div style="position:relative;height:280px">
<canvas id="chartModel"></canvas>
</div>
<div id="statsByModel" style="margin-top:12px"></div>
</div>
<div class="card">
<div class="card-title">最近请求日志 <button class="btn btn-sm btn-danger" style="float:right" onclick="clearLogs()">清空日志</button></div>
<div id="recentLogs"></div>
<div id="logPagination" style="display:flex;align-items:center;justify-content:center;gap:8px;margin-top:12px"></div>
</div>
</div>

<!-- 设置 -->
<div class="panel" id="panel-settings">
<div class="card">
<div class="card-title">全局设置</div>
<div class="form-row">
<div class="form-group">
<label>输入预算预设 (回车添加，或逗号分隔添加多个)</label>
<div class="tag-input-wrap" id="inputPresetsWrap">
<input class="tag-input" id="inputPresetInput" placeholder="如: 32K" oninput="this.value=this.value.trim()" onkeydown="handlePresetKeydown(event,'input')">
</div>
</div>
<div class="form-group">
<label>输出预算预设 (回车添加，或逗号分隔添加多个)</label>
<div class="tag-input-wrap" id="outputPresetsWrap">
<input class="tag-input" id="outputPresetInput" placeholder="如: 8K" oninput="this.value=this.value.trim()" onkeydown="handlePresetKeydown(event,'output')">
</div>
</div>
</div>
<button class="btn btn-primary" onclick="saveSettings()">保存设置</button>
</div>
<div class="card">
<div class="card-title">模型上下文配置 <button class="btn btn-primary" onclick="showGlobalModelConfigModal()">+ 添加配置</button></div>
<p style="color:var(--muted);font-size:13px;margin-bottom:16px">为指定模型配置输入/输出上下文预算。配置后网关转发时自动覆盖 max_tokens，留空则走客户端值。</p>
<div id="globalModelConfigList"></div>
</div>
</div>

<!-- 聊天测试 -->
<div class="panel" id="panel-chat">
<div class="card" style="display:flex;flex-direction:column;height:calc(100vh - 200px);min-height:500px">
<div class="card-title" style="display:flex;align-items:center;gap:8px;flex-wrap:wrap">
<span>聊天测试</span>
<select class="select" id="chatProvider" onchange="onChatProviderChange()" style="width:auto"></select>
<select class="select" id="chatModel" style="width:auto"></select>
<button class="btn btn-outline" onclick="clearChat()">清空</button>
</div>
<div id="chatMessages" style="flex:1;overflow-y:auto;padding:12px;background:var(--bg);border:1px solid var(--border);border-radius:var(--radius);margin-bottom:12px"></div>
<div style="display:flex;gap:8px">
<textarea class="input" id="chatInput" placeholder="输入消息，Enter发送，Shift+Enter换行..." style="flex:1;resize:none;height:60px" onkeydown="handleChatKeydown(event)"></textarea>
<button class="btn btn-primary" onclick="sendChatMessage()" id="chatSendBtn">发送</button>
</div>
</div>
</div>

<!-- 接入配置 -->
<div class="panel" id="panel-config">
<div class="card">
<div class="card-title">客户端接入配置</div>
<p style="color:var(--muted);font-size:13px;margin-bottom:12px">在 Trae / WorkBuddy / Claude Code 等工具中使用以下配置：</p>
<div class="config-box">
<h4>OpenAI 格式 (Trae / WorkBuddy)</h4>
<div class="config-line">Base URL: <span id="cfgOpenAIUrl">http://127.0.0.1:3458/v1</span> <button class="copy-btn" onclick="copyText(document.getElementById('cfgOpenAIUrl').textContent,this)">复制</button></div>
<div class="config-line">API Key: <span id="cfgOpenAIKey">在 API Keys 页面生成</span></div>
<div class="config-line">Model: <span id="cfgModel">all</span> <button class="copy-btn" onclick="copyText(document.getElementById('cfgModel').textContent,this)">复制</button></div>
</div>
<div class="config-box">
<h4>Anthropic 格式 (Claude Code / Cline)</h4>
<div class="config-line">Base URL: <span id="cfgAnthropicUrl">http://127.0.0.1:3458</span> <button class="copy-btn" onclick="copyText(document.getElementById('cfgAnthropicUrl').textContent,this)">复制</button></div>
<div class="config-line">API Key: <span id="cfgAnthropicKey">在 API Keys 页面生成</span></div>
<div class="config-line">Model: <span id="cfgModel2">all</span> <button class="copy-btn" onclick="copyText(document.getElementById('cfgModel2').textContent,this)">复制</button></div>
</div>
<div class="config-box">
<h4>指定站点调用 (URL 路径)</h4>
<p style="color:var(--muted);font-size:12px;margin-bottom:6px">在路径中加入 /p/{站点ID} 即可指定站点，模型在请求体中指定：</p>
<div class="config-line">OpenAI: <span id="cfgPOpenAI">/v1/chat/completions/p/{站点ID}</span> <button class="copy-btn" onclick="copyText(document.getElementById('cfgPOpenAI').textContent,this)">复制</button></div>
<div class="config-line">Anthropic: <span id="cfgPAnthropic">/v1/messages/p/{站点ID}</span> <button class="copy-btn" onclick="copyText(document.getElementById('cfgPAnthropic').textContent,this)">复制</button></div>
</div>
<div class="config-box">
<h4>健康检查</h4>
<div class="config-line">GET <span id="cfgHealthUrl">http://127.0.0.1:3458/health</span></div>
</div>
</div>
</div>
</div>

<!-- Provider Modal -->
<div class="modal-overlay" id="providerModal">
<div class="modal">
<div class="modal-title" id="providerModalTitle">添加中转站</div>
<input type="hidden" id="provId">
<div class="form-group">
<label>名称</label>
<input class="input" id="provName" placeholder="如: OpenAI官方 / 某中转站">
</div>
<div class="form-row">
<div class="form-group">
<label>协议格式</label>
<select class="select" id="provFormat">
<option value="openai">OpenAI (Chat Completions)</option>
<option value="anthropic">Anthropic (Messages)</option>
</select>
</div>
<div class="form-group">
<label>状态</label>
<select class="select" id="provStatus">
<option value="active">启用</option>
<option value="disabled">禁用</option>
</select>
</div>
</div>
<div class="form-group">
<label>Base URL</label>
<input class="input" id="provBaseUrl" placeholder="https://api.openai.com/v1" oninput="this.value=this.value.trim()">
</div>
<div class="form-group">
<label>代理设置</label>
<div style="display:flex;align-items:center;gap:12px;flex-wrap:wrap">
<label style="display:flex;align-items:center;gap:6px;cursor:pointer;font-size:13px;color:var(--text);margin:0">
<input type="checkbox" id="provProxyEnabled" onchange="toggleProxyFields()"> 启用代理
</label>
<select class="select" id="provProxyType" style="width:auto;display:none">
<option value="http">HTTP</option>
<option value="https">HTTPS</option>
<option value="socks5">SOCKS5</option>
</select>
<input class="input" id="provProxyAddr" placeholder="127.0.0.1:7890" style="flex:1;min-width:150px;display:none" oninput="this.value=this.value.trim()">
</div>
</div>
<div class="form-group">
<label>打卡签到地址 (可选，支持签到送积分的站点)</label>
<input class="input" id="provCheckinUrl" placeholder="https://api.xxx.com/user/checkin" oninput="this.value=this.value.trim()">
</div>
<div class="form-group">
<label>API Keys (多Key轮询，回车或逗号分隔添加多个)</label>
<div id="provKeysList" style="margin-bottom:8px"></div>
<div style="display:flex;gap:8px;margin-bottom:8px">
<input class="input" id="provKeyInput" placeholder="sk-... 回车或逗号分隔添加多个" style="flex:1" oninput="this.value=this.value.trim()" onkeydown="handleKeyKeydown(event)">
<input class="input" id="provKeyName" placeholder="备注(可选)" style="width:120px">
<button class="btn btn-outline" onclick="addProvKey()">添加</button>
<button class="btn btn-outline" onclick="clearProvKeys()">🗑 一键清空</button>
</div>
<div style="display:flex;gap:8px;align-items:center">
<label style="font-size:13px;color:var(--muted);display:flex;align-items:center;gap:4px;cursor:pointer"><input type="checkbox" id="provKeySelectAll" onchange="toggleAllKeys(this.checked)"> 全选</label>
<span class="badge badge-active" id="provKeySelCount">已选 0</span>
<button class="btn btn-outline btn-sm" onclick="batchDeleteKeys()">🗑 批量删除</button>
<button class="btn btn-outline btn-sm" onclick="testAllProvKeys()">⚡ 一键检测</button>
</div>
</div>
<div class="form-group">
<label>支持的模型 (回车添加，或逗号分隔添加多个)</label>
<div style="display:flex;gap:8px;margin-bottom:8px;flex-wrap:wrap">
<button class="btn btn-outline" id="fetchModelsBtn" onclick="fetchProviderModels()" style="font-size:13px">🔌 一键获取模型</button>
<button class="btn btn-outline" onclick="clearProviderModels()" style="font-size:13px">🗑 一键清空</button>
<span id="fetchModelsStatus" style="color:var(--muted);font-size:12px;align-self:center"></span>
</div>
<div class="tag-input-wrap" id="provModelsWrap">
<input class="tag-input" id="provModelInput" placeholder="输入模型名后回车..." oninput="this.value=this.value.trim()" onkeydown="handleModelKeydown(event)">
</div>
</div>
<div class="form-group">
<label>自定义请求头 (回车添加；支持 Key: Value 或 Key=Value，分号分隔多个)</label>
<div id="provHeadersList" style="margin-bottom:6px"></div>
<div style="display:flex;gap:8px">
<input class="input" id="provHeaderInput" placeholder="Authorization: Bearer xxx; Content-Type: application/json" style="flex:1" oninput="this.value=this.value.trim()" onkeydown="handleHeaderKeydown(event)">
<button class="btn btn-outline" onclick="addProvHeader()">添加</button>
<button class="btn btn-outline" onclick="clearProvHeaders()">🗑 一键清空</button>
</div>
</div>
<div class="form-group">
<button class="btn btn-outline" onclick="testProvider()" id="testBtn">测试连接</button>
<div id="testResult"></div>
</div>
<div class="modal-actions">
<button class="btn btn-outline" onclick="closeModal('providerModal')">取消</button>
<button class="btn btn-primary" onclick="saveProvider()">保存</button>
</div>
</div>
</div>

<!-- OpenCode Import Modal -->
<div class="modal-overlay" id="opencodeModal">
<div class="modal">
<div class="modal-title">导入 OpenCode 配置</div>
<p style="color:var(--muted);font-size:13px;margin-bottom:8px">粘贴 OpenCode 的 JSON 配置（含 provider.openai/anthropic），或上传文件：</p>
<textarea class="input" id="opencodeInput" placeholder='{"provider":{"openai":{"options":{"baseURL":"https://...","apiKey":"sk-..."},"models":{...}}}}' style="width:100%;min-height:280px;resize:vertical;font-family:'Courier New',monospace;font-size:12px"></textarea>
<div style="display:flex;gap:8px;margin-top:8px">
<button class="btn btn-outline" onclick="document.getElementById('opencodeFile').click()">📁 上传文件</button>
<input type="file" id="opencodeFile" accept=".json,application/json" style="display:none" onchange="handleOpenCodeFile(event)">
</div>
<div class="modal-actions">
<button class="btn btn-outline" onclick="closeModal('opencodeModal')">取消</button>
<button class="btn btn-primary" onclick="submitOpenCode()">导入</button>
</div>
</div>
</div>

<!-- Import Config Modal -->
<div class="modal-overlay" id="importModal">
<div class="modal">
<div class="modal-title">导入配置</div>
<p style="color:var(--muted);font-size:13px;margin-bottom:8px">粘贴 JSON 配置（中转站配置或 newapi 连接格式），或上传文件：</p>
<textarea class="input" id="importInput" placeholder='{"_type":"newapi_channel_conn","key":"sk-...","url":"https://..."}' style="width:100%;min-height:280px;resize:vertical;font-family:'Courier New',monospace;font-size:12px"></textarea>
<div style="display:flex;gap:8px;margin-top:8px">
<button class="btn btn-outline" onclick="document.getElementById('importFile').click()">📁 上传文件</button>
<input type="file" id="importFile" accept=".json,application/json" style="display:none" onchange="handleImportFile(event)">
</div>
<div class="modal-actions">
<button class="btn btn-outline" onclick="closeModal('importModal')">取消</button>
<button class="btn btn-primary" onclick="submitImport()">导入</button>
</div>
</div>
</div>

<!-- Model Test Modal -->
<div class="modal-overlay" id="modelTestModal">
<div class="modal">
<div class="modal-title">模型对话测试</div>
<div id="modelTestContent"><div class="empty">测试中...</div></div>
<div class="modal-actions">
<button class="btn btn-outline" onclick="closeModal('modelTestModal')">关闭</button>
</div>
</div>
</div>

<!-- Model Config Modal -->
<div class="modal-overlay" id="modelConfigModal">
<div class="modal">
<div class="modal-title">模型上下文配置</div>
<input type="hidden" id="mcProviderId">
<input type="hidden" id="mcModel">
<div class="form-group">
<label>模型</label>
<div class="mono" id="mcModelName" style="padding:8px;background:var(--bg);border-radius:4px"></div>
</div>
<div class="form-row">
<div class="form-group">
<label>输入上下文预算</label>
<select class="select" id="mcInputLimit"></select>
</div>
<div class="form-group">
<label>输出预算</label>
<select class="select" id="mcOutputLimit"></select>
</div>
</div>
<p style="color:var(--muted);font-size:12px;margin-bottom:12px">配置后网关会覆盖请求中的 max_tokens 为输出预算值。留空则不干预，走客户端传的值。</p>
<div class="modal-actions">
<button class="btn btn-outline" onclick="closeModal('modelConfigModal')">取消</button>
<button class="btn btn-primary" onclick="saveModelConfig()">保存</button>
</div>
</div>
</div>

<!-- Global Model Config Modal -->
<div class="modal-overlay" id="globalModelConfigModal">
<div class="modal">
<div class="modal-title" id="globalMcTitle">添加模型上下文配置</div>
<div class="form-group">
<label>站点</label>
<select class="select" id="gmcProvider" onchange="onGmcProviderChange()"></select>
</div>
<div class="form-group">
<label>模型</label>
<select class="select" id="gmcModel"></select>
</div>
<div class="form-row">
<div class="form-group">
<label>输入上下文预算</label>
<select class="select" id="gmcInputLimit"></select>
</div>
<div class="form-group">
<label>输出预算</label>
<select class="select" id="gmcOutputLimit"></select>
</div>
</div>
<p style="color:var(--muted);font-size:12px;margin-bottom:12px">配置后网关会覆盖请求中的 max_tokens 为输出预算值。留空则不干预，走客户端传的值。</p>
<div class="modal-actions">
<button class="btn btn-outline" onclick="closeModal('globalModelConfigModal')">取消</button>
<button class="btn btn-primary" onclick="saveGlobalModelConfig()">保存</button>
</div>
</div>
</div>

<!-- Alias Modal -->
<div class="modal-overlay" id="aliasModal">
<div class="modal">
<div class="modal-title" id="aliasModalTitle">添加路由别名</div>
<input type="hidden" id="aliasId">
<div class="form-group">
<label>别名 (客户端请求时用的模型名)</label>
<input class="input" id="aliasName" placeholder="如: workbuddy / default / my-model">
</div>
<div class="form-group">
<label>目标站点</label>
<select class="select" id="aliasProvider" onchange="updateAliasModels()"></select>
</div>
<div class="form-group">
<label>实际模型</label>
<select class="select" id="aliasModel"></select>
</div>
<div class="modal-actions">
<button class="btn btn-outline" onclick="closeModal('aliasModal')">取消</button>
<button class="btn btn-primary" onclick="saveAlias()">保存</button>
</div>
</div>
</div>

<!-- Key Modal -->
<div class="modal-overlay" id="keyModal">
<div class="modal">
<div class="modal-title">生成 API Key</div>
<div class="form-group">
<label>名称 (备注)</label>
<input class="input" id="keyName" placeholder="如: Trae使用 / WorkBuddy使用">
</div>
<div class="modal-actions">
<button class="btn btn-outline" onclick="closeModal('keyModal')">取消</button>
<button class="btn btn-primary" onclick="createKey()">生成</button>
</div>
</div>
</div>

<div id="toast" style="display:none"></div>

<script>
const API='/admin/api';
let activeProviderId='';
let editingModels=[];
let editingKeys=[];
let allProvidersCache=[];
let inputPresets=['32K','64K','128K','256K','384K','512K','1M'];
let outputPresets=['8K','16K','32K','64K','128K','256K','384K'];

// --- 主题切换 ---
function initTheme(){
const saved=localStorage.getItem('aim-theme')||'light';
document.documentElement.setAttribute('data-theme',saved);
updateThemeBtn(saved);
}
function toggleTheme(){
const cur=document.documentElement.getAttribute('data-theme');
const next=cur==='dark'?'light':'dark';
document.documentElement.setAttribute('data-theme',next);
localStorage.setItem('aim-theme',next);
updateThemeBtn(next);
}
function updateThemeBtn(t){
document.getElementById('themeBtn').textContent=t==='dark'?'☀️':'🌙';
document.getElementById('themeBtn').title=t==='dark'?'切换到白天模式':'切换到夜间模式';
}

function switchTab(e,name){
document.querySelectorAll('.tab').forEach(t=>t.classList.remove('active'));
document.querySelectorAll('.panel').forEach(p=>p.classList.remove('active'));
e.target.classList.add('active');
document.getElementById('panel-'+name).classList.add('active');
if(name==='stats')loadStats();
if(name==='config')loadConfig();
if(name==='settings')loadSettings();
if(name==='aliases')loadAliases();
if(name==='chat')initChat();
}

// --- 聊天测试 ---
let chatHistory=[];
let chatInitialized=false;
let chatGatewayKey='';

async function initChat(){
if(!chatInitialized){
chatInitialized=true;
document.getElementById('chatMessages').innerHTML='<div class="empty" style="text-align:center;padding:40px">选择站点、模型和Key，输入消息开始测试</div>';
}
await loadChatProviders();
// 加载网关API Key（用于鉴权）
try{
const res=await fetch(API+'/keys');
const data=await res.json();
const activeKey=(data||[]).find(k=>k.status==='active');
chatGatewayKey=activeKey?activeKey.key:'';
}catch(e){}
}

async function loadChatProviders(){
const res=await fetch(API+'/providers');
const data=await res.json();
allProvidersCache=data||[];
const sel=document.getElementById('chatProvider');
const curVal=sel.value;
sel.innerHTML='';
for(const p of data||[]){
if(p.status==='active'){
sel.innerHTML+='<option value="'+p.id+'">'+esc(p.name)+' ('+p.format+')</option>';
}
}
if(curVal)sel.value=curVal;
onChatProviderChange();
}

function onChatProviderChange(){
const pid=document.getElementById('chatProvider').value;
const p=allProvidersCache.find(x=>x.id===pid);
// 填充模型
const modelSel=document.getElementById('chatModel');
modelSel.innerHTML='';
if(p){
const disabledSet=new Set(p.disabledModels||[]);
for(const m of(p.models||[])){
if(!disabledSet.has(m))modelSel.innerHTML+='<option value="'+esc(m)+'">'+esc(m)+'</option>';
}
}
}

function clearChat(){
chatHistory=[];
document.getElementById('chatMessages').innerHTML='<div class="empty" style="text-align:center;padding:40px">对话已清空</div>';
}

function handleChatKeydown(e){
if(e.key==='Enter'&&!e.shiftKey){
e.preventDefault();
sendChatMessage();
}
}

function appendChatMsg(role,content){
const el=document.getElementById('chatMessages');
// 清除空状态提示
if(el.querySelector('.empty'))el.innerHTML='';
const div=document.createElement('div');
div.className='chat-msg '+(role==='error'?'error':role);
div.innerHTML='<div class="role">'+(role==='user'?'你':role==='assistant'?'AI':'错误')+'</div>'+esc(content);
el.appendChild(div);
el.scrollTop=el.scrollHeight;
return div;
}

async function sendChatMessage(){
const input=document.getElementById('chatInput');
const msg=input.value.trim();
if(!msg)return;
const pid=document.getElementById('chatProvider').value;
const model=document.getElementById('chatModel').value;
if(!pid||!model){
toast('请选择站点和模型','error');return;
}
input.value='';
appendChatMsg('user',msg);
chatHistory.push({role:'user',content:msg});
// 添加AI占位
const aiDiv=appendChatMsg('assistant','');
aiDiv.querySelector('.role').textContent='AI 正在思考...';
document.getElementById('chatSendBtn').disabled=true;
try{
// 通过网关中转请求，指定 provider
const gatewayUrl=location.origin+'/v1/chat/completions/p/'+pid;
const headers={'Content-Type':'application/json'};
if(chatGatewayKey)headers['Authorization']='Bearer '+chatGatewayKey;
const body={model:model,messages:chatHistory.map(m=>({role:m.role,content:m.content})),stream:true};
const res=await fetch(gatewayUrl,{method:'POST',headers:headers,body:JSON.stringify(body)});
if(!res.ok){
const errText=await res.text();
aiDiv.className='chat-msg error';
aiDiv.innerHTML='<div class="role">错误</div>HTTP '+res.status+': '+errText;
chatHistory.pop();
return;
}
// 流式读取
const reader=res.body.getReader();
const decoder=new TextDecoder();
let fullContent='';
let buffer='';
while(true){
const{done,value}=await reader.read();
if(done)break;
buffer+=decoder.decode(value,{stream:true});
const lines=buffer.split('\n');
buffer=lines.pop()||'';
for(const line of lines){
const trimmed=line.trim();
if(!trimmed||!trimmed.startsWith('data:'))continue;
const data=trimmed.slice(5).trim();
if(data==='[DONE]')continue;
try{
const json=JSON.parse(data);
if(json.choices&&json.choices[0]&&json.choices[0].delta&&json.choices[0].delta.content){
fullContent+=json.choices[0].delta.content;
aiDiv.innerHTML='<div class="role">AI</div>'+esc(fullContent);
document.getElementById('chatMessages').scrollTop=document.getElementById('chatMessages').scrollHeight;
}
}catch(e){}
}
}
if(!fullContent){
// 非流式响应
const json=await res.json();
if(json.choices&&json.choices[0]&&json.choices[0].message&&json.choices[0].message.content){
fullContent=json.choices[0].message.content;
}
aiDiv.innerHTML='<div class="role">AI</div>'+esc(fullContent||'(空回复)');
}
chatHistory.push({role:'assistant',content:fullContent});
}catch(e){
aiDiv.className='chat-msg error';
aiDiv.innerHTML='<div class="role">错误</div>'+esc(e.message);
chatHistory.pop();
}finally{
document.getElementById('chatSendBtn').disabled=false;
}
}

function toggleProxyFields(){
const enabled=document.getElementById('provProxyEnabled').checked;
document.getElementById('provProxyType').style.display=enabled?'':'none';
document.getElementById('provProxyAddr').style.display=enabled?'':'none';
}
function renderModelTags(){
const wrap=document.getElementById('provModelsWrap');
const input=document.getElementById('provModelInput');
// 清除旧标签
wrap.querySelectorAll('.tag').forEach(t=>t.remove());
editingModels.forEach((m,i)=>{
const tag=document.createElement('span');
tag.className='tag';
tag.innerHTML=esc(m)+'<button class="tag-x" onclick="removeModelTag('+i+')">×</button>';
wrap.insertBefore(tag,input);
});
}
function addModelTag(val){
val=(val||'').trim();
if(!val)return;
// 支持逗号分隔多个
const parts=val.split(',').map(s=>s.trim()).filter(s=>s);
for(const p of parts){
if(p&&!editingModels.includes(p))editingModels.push(p);
}
document.getElementById('provModelInput').value='';
renderModelTags();
}
function removeModelTag(i){
editingModels.splice(i,1);
renderModelTags();
}
function handleModelKeydown(e){
if(e.key==='Enter'||e.key===','){
e.preventDefault();
addModelTag(e.target.value);
}else if(e.key==='Backspace'&&e.target.value===''){
editingModels.pop();
renderModelTags();
}
}

// --- 多 Key 管理 ---
let adminKeySelected=new Set();
function renderKeyList(){
const el=document.getElementById('provKeysList');
el.innerHTML='';
if(editingKeys.length===0){
adminKeySelected.clear();
updateKeySelUI();
el.innerHTML='<div style="color:var(--muted);font-size:12px;padding:4px 0">暂无 Key，请在下方添加</div>';
return;
}
const baseUrl=document.getElementById('provBaseUrl').value;
const format=document.getElementById('provFormat').value;
const firstModel=editingModels[0]||'gpt-4o-mini';
editingKeys.forEach((k,i)=>{
const item=document.createElement('div');
item.className='key-item';
const isActive=(k.status||'active')==='active';
const checked=adminKeySelected.has(i)?'checked':'';
let html='<input type="checkbox" '+checked+' onchange="toggleKeySelect('+i+',this.checked)" title="选择" style="flex-shrink:0">';
html+='<input class="input key-edit-val" value="'+esc(k.key)+'" oninput="this.value=this.value.trim();editingKeys['+i+'].key=this.value" style="flex:1;font-size:12px;padding:4px 8px" title="可编辑">';
html+='<input class="input key-edit-name" value="'+esc(k.name||'')+'" placeholder="备注" oninput="editingKeys['+i+'].name=this.value" style="width:100px;font-size:12px;padding:4px 8px">';
html+='<span class="badge badge-'+(k.status||'active')+'">'+(k.status||'active')+'</span>';
html+='<button class="copy-btn" onclick="toggleProvKey('+i+')" title="'+(isActive?'禁用':'启用')+'">'+(isActive?'⏸️':'▶️')+'</button>';
html+='<button class="copy-btn" onclick="testProvKey('+i+')">测试</button>';
html+='<button class="copy-btn" onclick="removeProvKey('+i+')" title="删除">×</button>';
item.innerHTML=html;
el.appendChild(item);
});
updateKeySelUI();
}
function toggleKeySelect(i,checked){
if(checked)adminKeySelected.add(i);else adminKeySelected.delete(i);
updateKeySelUI();
}
function toggleAllKeys(checked){
adminKeySelected.clear();
if(checked){editingKeys.forEach((_,i)=>adminKeySelected.add(i));}
renderKeyList();
}
function updateKeySelUI(){
const all=document.getElementById('provKeySelectAll');
const cnt=document.getElementById('provKeySelCount');
if(all)all.checked=editingKeys.length>0&&adminKeySelected.size===editingKeys.length;
if(cnt)cnt.textContent='已选 '+adminKeySelected.size;
}
function batchDeleteKeys(){
if(adminKeySelected.size===0){toast('未选择任何 Key','error');return;}
if(!confirm('确定删除选中的 '+adminKeySelected.size+' 个 Key 吗？'))return;
const idx=[...adminKeySelected].sort((a,b)=>b-a);
idx.forEach(i=>editingKeys.splice(i,1));
adminKeySelected.clear();
renderKeyList();
toast('已批量删除 '+idx.length+' 个 Key','success');
}
async function testAllProvKeys(){
const baseUrl=document.getElementById('provBaseUrl').value;
const format=document.getElementById('provFormat').value;
const model=editingModels[0]||'gpt-4o-mini';
if(!baseUrl){toast('请先填写 Base URL','error');return;}
if(editingKeys.length===0){toast('没有 Key 可检测','error');return;}
const modal=document.getElementById('modelTestModal');
const content=document.getElementById('modelTestContent');
modal.classList.add('show');
content.innerHTML='<div class="empty"><span class="loading"></span> 正在检测 '+editingKeys.length+' 个 Key ...</div>';
let ok=0,fail=0,failList=[];
for(let i=0;i<editingKeys.length;i++){
const k=editingKeys[i];
content.innerHTML='<div class="empty">正在检测 <strong>'+esc(k.key.substring(0,8)+'...')+'</strong> ('+(i+1)+'/'+editingKeys.length+')<br><div class="loading" style="margin-top:8px"></div></div>';
try{
const res=await fetch(API+'/test',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({
baseUrl:baseUrl,apiKey:k.key,format:format,model:model,
customHeaders:{...editingHeaders},
proxyEnabled:document.getElementById('provProxyEnabled').checked,
proxyType:document.getElementById('provProxyType').value,
proxyAddr:document.getElementById('provProxyAddr').value.trim()
})});
const data=await res.json();
if(data.success){ok++;}else{fail++;failList.push({key:k.key,reason:data.error||'失败'});}
}catch(e){fail++;failList.push({key:k.key,reason:e.message});}
}
let html='<div style="margin-bottom:12px"><strong>检测完成</strong> (HTTP '+editingKeys.length+' 个)</div>';
html+='<div style="margin-bottom:8px"><span class="badge badge-active">可用 '+ok+'</span> <span class="badge badge-disabled" style="margin-left:4px">不可用 '+fail+'</span></div>';
if(failList.length){
html+='<div style="font-size:13px;color:var(--muted);margin-bottom:4px">不可用明细</div>';
failList.forEach(f=>{
html+='<div class="test-error" style="padding:8px;margin-bottom:6px;font-size:12px"><span class="mono">'+esc(f.key.substring(0,12)+'...')+'</span><br>'+esc(f.reason)+'</div>';
});
}
content.innerHTML=html;
}
function addProvKey(){
const keyInput=document.getElementById('provKeyInput');
const nameInput=document.getElementById('provKeyName');
const raw=keyInput.value.trim();
if(!raw){toast('请输入 Key','error');return;}
const name=nameInput.value.trim();
const parts=(raw.includes(',')||raw.includes('，'))?raw.split(/[,，]/):[raw];
let added=0;
parts.forEach(p=>{
p=p.trim();
if(!p)return;
editingKeys.push({id:'',key:p,name:name,status:'active'});
added++;
});
keyInput.value='';
nameInput.value='';
renderKeyList();
keyInput.focus();
toast('已添加 '+added+' 个 Key','success');
}
function clearProvKeys(){
if(!editingKeys.length){toast('没有可清空的 Key','error');return;}
if(!confirm('确定清空全部 Key 吗？'))return;
editingKeys=[];
adminKeySelected.clear();
renderKeyList();
toast('已清空全部 Key','success');
}
function removeProvKey(i){
editingKeys.splice(i,1);
const shifted=new Set();
adminKeySelected.forEach(idx=>{
if(idx===i)return;
if(idx>i)shifted.add(idx-1);else shifted.add(idx);
});
adminKeySelected=shifted;
renderKeyList();
}
function toggleProvKey(i){
const k=editingKeys[i];
k.status=(k.status||'active')==='active'?'disabled':'active';
renderKeyList();
}
function handleKeyKeydown(e){
if(e.key==='Enter'){e.preventDefault();addProvKey();}
}

// 测试单个 Key
async function testProvKey(i){
const k=editingKeys[i];
const baseUrl=document.getElementById('provBaseUrl').value;
const format=document.getElementById('provFormat').value;
const model=editingModels[0]||'gpt-4o-mini';
if(!baseUrl||!k.key){toast('请先填写 Base URL 和 Key','error');return;}
const modal=document.getElementById('modelTestModal');
const content=document.getElementById('modelTestContent');
modal.classList.add('show');
content.innerHTML='<div class="empty"><span class="loading"></span> 正在测试 Key '+esc(k.key.substring(0,8)+'...')+' ...</div>';
try{
const res=await fetch(API+'/test',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({
baseUrl:baseUrl,apiKey:k.key,format:format,model:model,
customHeaders:{...editingHeaders},
proxyEnabled:document.getElementById('provProxyEnabled').checked,
proxyType:document.getElementById('provProxyType').value,
proxyAddr:document.getElementById('provProxyAddr').value.trim()
})});
const data=await res.json();
let html='<div style="margin-bottom:12px">';
html+='<div style="font-size:13px;color:var(--muted);margin-bottom:4px">测试 Key</div>';
html+='<div class="mono" style="margin-bottom:4px;word-break:break-all">'+esc(k.key)+'</div>';
if(k.name){html+='<div style="font-size:12px;color:var(--muted);margin-bottom:4px">备注: '+esc(k.name)+'</div>';}
html+='<div style="font-size:13px;color:var(--muted);margin-bottom:4px">测试模型</div>';
html+='<div class="mono" style="margin-bottom:8px">'+esc(model)+'</div>';
html+='<div style="font-size:13px;color:var(--muted);margin-bottom:4px">请求地址</div>';
html+='<div class="mono" style="margin-bottom:8px;font-size:12px;word-break:break-all">'+esc(data.testUrl||'-')+'</div>';
html+='<div style="font-size:13px;color:var(--muted);margin-bottom:4px">发送消息</div>';
html+='<div class="mono" style="margin-bottom:8px;padding:8px;background:var(--bg);border-radius:4px">'+esc(data.testMessage||'-')+'</div>';
html+='</div>';
if(data.success){
html+='<div class="test-success" style="margin-bottom:8px"><strong>Key 可用</strong> (HTTP '+data.status+')</div>';
html+='<div style="font-size:13px;color:var(--muted);margin-bottom:4px">AI 回复</div>';
html+='<div class="test-success" style="padding:12px;margin-bottom:8px">'+esc(data.content||'(空)')+'</div>';
}else{
html+='<div class="test-error" style="margin-bottom:8px"><strong>Key 不可用</strong> (HTTP '+data.status+')</div>';
html+='<div class="test-error" style="padding:12px;margin-bottom:8px">'+esc(data.error||'未知错误')+'</div>';
}
if(data.raw){
html+='<details style="margin-top:8px"><summary style="cursor:pointer;font-size:12px;color:var(--muted)">原始响应</summary><pre class="mono" style="font-size:11px;padding:8px;background:var(--bg);border-radius:4px;overflow-x:hidden;white-space:pre-wrap;word-break:break-word;overflow-wrap:anywhere;margin-top:4px">'+esc(data.raw)+'</pre></details>';
}
content.innerHTML=html;
}catch(e){
content.innerHTML='<div class="test-error">请求失败: '+esc(e.message)+'</div>';
}
}

// --- Providers ---
let adminProvSelected=new Set();
function toggleProviderSelect(id,checked){
if(checked)adminProvSelected.add(id);else adminProvSelected.delete(id);
updateProvSelUI();
}
function toggleAllProviders(checked){
adminProvSelected.clear();
if(checked){allProvidersCache.forEach(p=>adminProvSelected.add(p.id));}
updateProvSelUI();
}
function updateProvSelUI(){
const all=document.getElementById('provSelectAll');
const cnt=document.getElementById('provSelCount');
if(all)all.checked=allProvidersCache.length>0&&adminProvSelected.size===allProvidersCache.length;
if(cnt)cnt.textContent='已选 '+adminProvSelected.size;
}
async function batchDeleteProviders(){
if(adminProvSelected.size===0){toast('未选择任何中转站','error');return;}
if(!confirm('确定删除选中的 '+adminProvSelected.size+' 个中转站吗？'))return;
let ok=0,fail=0;
for(const id of adminProvSelected){
const res=await fetch(API+'/providers/'+id,{method:'DELETE'});
if(res.ok)ok++;else fail++;
}
adminProvSelected.clear();
toast('已删除 '+ok+' 个，失败 '+fail+' 个','success');
loadProviders();
}
// 设置中转站状态 (active/disabled)，复用完整 provider 对象
async function setProviderStatus(id,status){
const p=allProvidersCache.find(x=>x.id===id);
if(!p)return false;
const body={...p,status:status};
const res=await fetch(API+'/providers/'+id,{method:'PUT',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});
return res.ok;
}
// 一键检测所有中转站是否正常，不正常自动改为不可用
async function testAllProviders(){
if(!allProvidersCache.length){toast('没有中转站可检测','error');return;}
const modal=document.getElementById('modelTestModal');
const content=document.getElementById('modelTestContent');
modal.classList.add('show');
content.innerHTML='<div class="empty"><span class="loading"></span> 正在检测 '+allProvidersCache.length+' 个中转站 ...</div>';
let ok=0,fail=0,needFix=[];
for(let i=0;i<allProvidersCache.length;i++){
const p=allProvidersCache[i];
content.innerHTML='<div class="empty">正在检测 <strong>'+esc(p.name)+'</strong> ('+(i+1)+'/'+allProvidersCache.length+')<br><div class="loading" style="margin-top:8px"></div></div>';
try{
const res=await fetch(API+'/providers/test/'+p.id,{method:'POST'});
const data=await res.json();
if(data&&data.success){ok++;}
else{fail++;needFix.push({id:p.id,name:p.name,reason:(data&&data.error)||'不可用'});}
}catch(e){fail++;needFix.push({id:p.id,name:p.name,reason:e.message});}
}
// 将检测失败的中转站状态改为不可用
let fixed=0;
for(const f of needFix){
if(f.id){if(await setProviderStatus(f.id,'disabled'))fixed++;}
}
let html='<div style="margin-bottom:12px"><strong>检测完成</strong> (共 '+allProvidersCache.length+' 个)</div>';
html+='<div style="margin-bottom:8px"><span class="badge badge-active">正常 '+ok+'</span> <span class="badge badge-disabled" style="margin-left:4px">异常 '+fail+'</span></div>';
if(fail>0)html+='<div style="font-size:13px;color:var(--muted);margin-bottom:4px">已自动将 '+fixed+' 个异常站点状态改为不可用</div>';
if(needFix.length){
html+='<div style="font-size:13px;color:var(--muted);margin-bottom:4px">异常明细</div>';
needFix.forEach(f=>{
html+='<div class="test-error" style="padding:8px;margin-bottom:6px;font-size:12px"><strong>'+esc(f.name)+'</strong><br>'+esc(f.reason)+'</div>';
});
}
content.innerHTML=html;
loadProviders();
}
async function loadProviders(){
const res=await fetch(API+'/providers');
const data=await res.json();
allProvidersCache=data||[];
const el=document.getElementById('providerList');
if(!data||data.length===0){
el.innerHTML='<div class="empty">暂无中转站，点击右上角添加</div>';
document.getElementById('headerInfo').textContent='未配置中转站';
return;
}
const sRes=await fetch(API+'/settings');
const settings=await sRes.json();
activeProviderId=settings.activeProviderId||'';
const defaultModel=settings.defaultModel||'';

let html='<table><tr><th style="width:30px"></th><th>名称</th><th>格式</th><th>Base URL</th><th>模型</th><th>Keys</th><th>状态</th><th>操作</th></tr>';
for(const p of data){
const isActive=p.id===activeProviderId;
const disabledSet=new Set(p.disabledModels||[]);
const enabledCount=(p.models||[]).filter(m=>!disabledSet.has(m)).length;
const totalCount=(p.models||[]).length;
const activeKeyCount=(p.apiKeys||[]).filter(k=>k.status==='active').length;
const totalKeyCount=(p.apiKeys||[]).length;
const checked=adminProvSelected.has(p.id)?'checked':'';
html+='<tr onclick="toggleModels(\''+p.id+'\')" style="cursor:pointer">';
html+='<td onclick="event.stopPropagation()"><input type="checkbox" '+checked+' onchange="toggleProviderSelect(\''+p.id+'\',this.checked)" title="选择"></td>';
html+='<td>'+esc(p.name)+(isActive?' <span class="badge badge-current">当前</span>':'')+'</td>';
html+='<td><span class="badge badge-'+p.format+'">'+p.format+'</span></td>';
html+='<td class="mono">'+esc(p.baseUrl)+'</td>';
html+='<td>'+enabledCount+'/'+totalCount+'</td>';
html+='<td>'+activeKeyCount+'/'+totalKeyCount+'</td>';
html+='<td><span class="badge badge-'+p.status+'">'+p.status+'</span></td>';
html+='<td style="white-space:nowrap" onclick="event.stopPropagation()">';
if(!isActive&&p.status==='active')html+='<button class="btn btn-sm btn-success" onclick="setActive(\''+p.id+'\')">启用</button> ';
if(p.checkinUrl){
const last=p.lastCheckin?fmtCheckin(p.lastCheckin):'';
html+='<button class="btn btn-sm btn-outline" onclick="doCheckin(\''+p.id+'\',this)" data-pid="'+p.id+'">'+(last?'已打卡 '+last:'打卡')+'</button> ';
}
html+='<button class="btn btn-sm btn-outline" onclick="editProvider(\''+p.id+'\')">编辑</button> ';
html+='<button class="btn btn-sm btn-outline" onclick="exportProvider(\''+p.id+'\')">导出</button> ';
html+='<button class="btn btn-sm btn-danger" onclick="deleteProvider(\''+p.id+'\')">删除</button>';
html+='</td>';
html+='</tr>';
// 模型展开行
html+='<tr id="models-'+p.id+'" style="display:none" onclick="event.stopPropagation()"><td colspan="8"><div class="model-row">';
for(const m of (p.models||[])){
const enabled=!disabledSet.has(m);
const isDefault=(m===p.defaultModel);
const modelUrl=getModelUrl(p,m);
const mc=(p.modelConfigs||[]).find(c=>c.model===m);
html+='<div class="model-chip'+(enabled?'':' disabled')+'">';
html+='<button class="chip-toggle" onclick="toggleModel(\''+p.id+'\',\''+esc(m)+'\')" title="'+(enabled?'点击禁用':'点击启用')+'">'+(enabled?'✅':'⏸️')+'</button>';
html+='<span class="chip-name">'+esc(m);
if(isDefault)html+=' <span class="badge badge-current">默认</span>';
if(mc&&mc.outputLimit)html+=' <span class="badge badge-openai" title="输出限制">'+esc(mc.outputLimit)+'</span>';
if(mc&&mc.inputLimit)html+=' <span class="badge badge-active" title="输入限制">'+esc(mc.inputLimit)+'</span>';
html+='</span>';
html+='<span class="chip-actions">';
html+='<span class="url-hint">'+modelUrl+'</span>';
html+='<button class="copy-btn" onclick="copyText(\''+esc(modelUrl)+'\',this)">复制</button>';
html+='<button class="copy-btn" onclick="testModel(\''+p.id+'\',\''+esc(m)+'\')">测试</button>';
html+='<button class="copy-btn" onclick="showModelConfigModal(\''+p.id+'\',\''+esc(m)+'\')">配置</button>';
if(!isDefault)html+='<button class="copy-btn" onclick="setModelDefault(\''+p.id+'\',\''+esc(m)+'\')">设为默认</button>';
html+='</span>';
html+='</div>';
}
if(!p.models||p.models.length===0)html+='<span class="empty" style="padding:8px">暂无模型</span>';
html+='</div></td></tr>';
}
html+='</table>';
el.innerHTML=html;
const active=data.find(p=>p.id===activeProviderId);
if(active){
document.getElementById('headerInfo').textContent='当前站点: '+active.name+' ('+active.format+')';
}else{
document.getElementById('headerInfo').textContent='未设置活跃站点';
}
}

function getModelUrl(p,model){
const base=location.protocol+'//'+location.host;
if(p.format==='anthropic'){
return base+'/v1/messages/p/'+p.id;
}
return base+'/v1/chat/completions/p/'+p.id;
}

function toggleModels(id){
const row=document.getElementById('models-'+id);
row.style.display=row.style.display==='none'?'table-row':'none';
}

async function toggleModel(providerId,model){
const url=API+'/providers/models/toggle/'+providerId+'?model='+encodeURIComponent(model);
const res=await fetch(url,{method:'PUT'});
if(res.ok){
const data=await res.json();
toast(data.enabled?'已启用: '+model:'已禁用: '+model,'success');
loadProviders();
}else{
toast('操作失败','error');
}
}

async function setModelDefault(providerId,model){
const url=API+'/providers/models/default/'+providerId+'?model='+encodeURIComponent(model);
const res=await fetch(url,{method:'PUT'});
if(res.ok){
toast('已设为站点默认模型: '+model,'success');
loadProviders();
}else{
toast('操作失败','error');
}
}

// 模型上下文配置
function showModelConfigModal(providerId,model){
const p=allProvidersCache.find(x=>x.id===providerId);
if(!p)return;
const mc=(p.modelConfigs||[]).find(c=>c.model===model)||{};
document.getElementById('mcProviderId').value=providerId;
document.getElementById('mcModel').value=model;
document.getElementById('mcModelName').textContent=model;
fillPresetSelect('mcInputLimit',mc.inputLimit||'',inputPresets);
fillPresetSelect('mcOutputLimit',mc.outputLimit||'',outputPresets);
document.getElementById('modelConfigModal').classList.add('show');
}

async function saveModelConfig(){
const providerId=document.getElementById('mcProviderId').value;
const model=document.getElementById('mcModel').value;
const body={
model:model,
inputLimit:document.getElementById('mcInputLimit').value,
outputLimit:document.getElementById('mcOutputLimit').value
};
const url=API+'/providers/models/config/'+providerId+'?model='+encodeURIComponent(model);
const res=await fetch(url,{method:'PUT',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});
if(res.ok){
closeModal('modelConfigModal');
toast('已保存模型配置','success');
loadProviders();
}else{
toast('保存失败','error');
}
}

function showProviderModal(){
document.getElementById('providerModalTitle').textContent='添加中转站';
document.getElementById('provId').value='';
document.getElementById('provName').value='';
document.getElementById('provFormat').value='openai';
document.getElementById('provStatus').value='active';
document.getElementById('provBaseUrl').value='';
document.getElementById('provCheckinUrl').value='';
document.getElementById('provProxyEnabled').checked=false;
document.getElementById('provProxyType').value='http';
document.getElementById('provProxyAddr').value='';
toggleProxyFields();
document.getElementById('provKeyInput').value='';
document.getElementById('provKeyName').value='';
editingKeys=[];
renderKeyList();
editingHeaders={};
renderHeaderList();
editingModels=[];
renderModelTags();
document.getElementById('testResult').innerHTML='';
document.getElementById('providerModal').classList.add('show');
}

async function editProvider(id){
const p=allProvidersCache.find(x=>x.id===id);
if(!p)return;
document.getElementById('providerModalTitle').textContent='编辑中转站';
document.getElementById('provId').value=p.id;
document.getElementById('provName').value=p.name;
document.getElementById('provFormat').value=p.format;
document.getElementById('provStatus').value=p.status;
document.getElementById('provBaseUrl').value=p.baseUrl;
document.getElementById('provCheckinUrl').value=p.checkinUrl||'';
document.getElementById('provProxyEnabled').checked=!!p.proxyEnabled;
document.getElementById('provProxyType').value=p.proxyType||'http';
document.getElementById('provProxyAddr').value=p.proxyAddr||'';
toggleProxyFields();
// 加载多 Key
editingKeys=(p.apiKeys||[]).map(k=>({...k}));
renderKeyList();
// 加载自定义请求头
editingHeaders={...((p.customHeaders)||{})};
renderHeaderList();
editingModels=[...(p.models||[])];
renderModelTags();
document.getElementById('testResult').innerHTML='';
document.getElementById('providerModal').classList.add('show');
}

async function saveProvider(){
const id=document.getElementById('provId').value;
// 合并输入框中未提交的内容
const inputVal=document.getElementById('provModelInput').value;
if(inputVal.trim())addModelTag(inputVal);
// 自定义请求头
const customHeaders={...editingHeaders};
const body={
name:document.getElementById('provName').value,
format:document.getElementById('provFormat').value,
status:document.getElementById('provStatus').value,
baseUrl:document.getElementById('provBaseUrl').value,
checkinUrl:document.getElementById('provCheckinUrl').value,
apiKey:editingKeys.length>0?editingKeys[0].key:'',
apiKeys:editingKeys.map(k=>({id:k.id||'',key:k.key,name:k.name||'',status:k.status||'active'})),
models:[...editingModels],
disabledModels:[],
customHeaders:customHeaders,
proxyEnabled:document.getElementById('provProxyEnabled').checked,
proxyType:document.getElementById('provProxyType').value,
proxyAddr:document.getElementById('provProxyAddr').value.trim()
};
if(!body.name||!body.baseUrl){
toast('请填写名称和 Base URL','error');return;
}
// 编辑时保留 disabledModels
if(id){
const old=allProvidersCache.find(x=>x.id===id);
if(old){
body.disabledModels=old.disabledModels||[];
body.defaultModel=old.defaultModel||'';
body.lastCheckin=old.lastCheckin||null;
}
}
const method=id?'PUT':'POST';
const url=id?API+'/providers/'+id:API+'/providers';
const res=await fetch(url,{method,headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});
if(res.ok){
closeModal('providerModal');
toast(id?'已更新':'已添加','success');
loadProviders();
}else{
toast('操作失败','error');
}
}

async function deleteProvider(id){
if(!confirm('确定删除此中转站？'))return;
const res=await fetch(API+'/providers/'+id,{method:'DELETE'});
if(res.ok){toast('已删除','success');loadProviders();}
}

async function setActive(id){
const res=await fetch(API+'/providers/active/'+id,{method:'PUT'});
if(res.ok){toast('已设为当前站点','success');loadProviders();}
}

function fmtCheckin(ts){
if(!ts)return'';
const d=new Date(ts);
const now=new Date();
const m=(d.getMonth()+1).toString().padStart(2,'0');
const day=d.getDate().toString().padStart(2,'0');
const hh=d.getHours().toString().padStart(2,'0');
const mm=d.getMinutes().toString().padStart(2,'0');
return m+'-'+day+' '+hh+':'+mm;
}

async function doCheckin(id,btn){
if(!btn)btn=document.querySelector('[data-pid="'+id+'"]');
const p=allProvidersCache.find(x=>x.id===id);
if(!p||!p.checkinUrl){toast('未配置打卡地址','error');return;}
// 新窗口打开签到链接
window.open(p.checkinUrl,'_blank');
// 记录打卡时间
const res=await fetch(API+'/providers/checkin/'+id,{method:'POST'});
const data=await res.json();
if(data.success){
toast('已记录打卡时间','success');
if(btn)btn.innerHTML='已打卡 '+fmtCheckin(new Date().toISOString());
loadProviders();
}else{
toast('记录失败: '+(data.message||''),'error');
}
}

async function testProvider(){
const btn=document.getElementById('testBtn');
const result=document.getElementById('testResult');
btn.innerHTML='<span class="loading"></span> 测试中...';
btn.disabled=true;
result.innerHTML='';
const inputVal=document.getElementById('provModelInput').value;
if(inputVal.trim())addModelTag(inputVal);
const body={
baseUrl:document.getElementById('provBaseUrl').value,
apiKey:editingKeys.length>0?editingKeys[0].key:'',
format:document.getElementById('provFormat').value,
model:editingModels[0]||'gpt-4o-mini',
customHeaders:{...editingHeaders},
proxyEnabled:document.getElementById('provProxyEnabled').checked,
proxyType:document.getElementById('provProxyType').value,
proxyAddr:document.getElementById('provProxyAddr').value.trim()
};
if(!body.baseUrl||!body.apiKey){
result.innerHTML='<div class="test-error">请填写 Base URL 和至少一个 API Key</div>';
btn.innerHTML='测试连接';btn.disabled=false;return;
}
try{
const res=await fetch(API+'/test',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});
const data=await res.json();
if(data.success){
result.innerHTML='<div class="test-success">连接成功! 响应: '+esc(data.content||'')+'</div>';
}else{
result.innerHTML='<div class="test-error">失败 (HTTP '+data.status+'): '+esc(data.error||'')+'</div>';
}
}catch(e){
result.innerHTML='<div class="test-error">请求失败: '+esc(e.message)+'</div>';
}
btn.innerHTML='测试连接';
btn.disabled=false;
}

// --- 一键获取模型 ---
async function fetchProviderModels(){
const btn=document.getElementById('fetchModelsBtn');
const status=document.getElementById('fetchModelsStatus');
const baseUrl=document.getElementById('provBaseUrl').value.trim();
const apiKey=editingKeys.length>0?editingKeys[0].key:'';
const format=document.getElementById('provFormat').value;
if(!baseUrl){
status.innerHTML='<span style="color:var(--red)">请先填写 Base URL</span>';return;
}
if(format!=='openai'){
status.innerHTML='<span style="color:var(--muted)">仅 OpenAI 格式支持一键获取</span>';return;
}
btn.innerHTML='<span class="loading"></span> 获取中...';
btn.disabled=true;
status.innerHTML='';
try{
const res=await fetch(API+'/providers/fetch-models',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({baseUrl,apiKey,format,customHeaders:{...editingHeaders},proxyEnabled:document.getElementById('provProxyEnabled').checked,proxyType:document.getElementById('provProxyType').value,proxyAddr:document.getElementById('provProxyAddr').value.trim()})});
const data=await res.json();
if(data.success){
const fetched=data.models||[];
const beforeCount=editingModels.length;
fetched.forEach(m=>{if(!editingModels.includes(m))editingModels.push(m);});
renderModelTags();
const added=editingModels.length-beforeCount;
status.innerHTML='<span style="color:var(--green)">✔ 获取 '+fetched.length+' 个模型，新增 '+added+' 个</span>';
if(fetched.length===0){
status.innerHTML='<span style="color:var(--muted)">接口返回成功但未解析到模型，请检查响应格式</span>';
}
}else{
status.innerHTML='<span style="color:var(--red)">✘ '+(data.error||'获取失败')+'</span>';
}
}catch(e){
status.innerHTML='<span style="color:var(--red)">✘ '+esc(e.message)+'</span>';
}
btn.innerHTML='🔌 一键获取模型';
btn.disabled=false;
}

function clearProviderModels(){
if(!confirm('确定清空该中转站的所有模型？'))return;
editingModels=[];
renderModelTags();
document.getElementById('fetchModelsStatus').innerHTML='';
toast('已清空所有模型','success');
}

// --- 自定义请求头编辑器 ---
let editingHeaders={};

function renderHeaderList(){
const el=document.getElementById('provHeadersList');
el.innerHTML='';
const keys=Object.keys(editingHeaders);
if(keys.length===0){
el.innerHTML='<div style="color:var(--muted);font-size:12px;padding:4px 0">暂无自定义请求头</div>';
return;
}
keys.forEach(k=>{
const item=document.createElement('div');
item.className='key-item';
item.innerHTML='<input class="input" value="'+esc(k)+'" style="flex:1;font-size:12px;padding:4px 8px" onchange="renameProvHeader(\''+esc(k)+'\',this.value)" placeholder="Key">';
item.innerHTML+='<input class="input" value="'+esc(editingHeaders[k])+'" style="flex:1;font-size:12px;padding:4px 8px" onchange="updateProvHeaderVal(\''+esc(k)+'\',this.value)" placeholder="Value">';
item.innerHTML+='<button class="key-x" onclick="removeProvHeader(\''+esc(k)+'\')" title="删除">×</button>';
el.appendChild(item);
});
}

function addProvHeader(){
const input=document.getElementById('provHeaderInput');
const raw=input.value.trim();
if(!raw){toast('请输入请求头','error');return;}
let added=0;
const parts=raw.split(/[;；]/);
parts.forEach(part=>{
part=part.trim();
if(!part){return;}
let sep=part.indexOf(':');
if(sep===-1)sep=part.indexOf('=');
if(sep===-1){toast('格式应为 Key: Value 或 Key=Value','error');return;}
const key=part.slice(0,sep).trim();
const val=part.slice(sep+1).trim();
if(!key){toast('请求头 Key 不能为空','error');return;}
editingHeaders[key]=val;
added++;
});
input.value='';
renderHeaderList();
input.focus();
toast('已添加 '+added+' 个请求头','success');
}

function clearProvHeaders(){
if(!Object.keys(editingHeaders).length){toast('没有可清空的请求头','error');return;}
editingHeaders={};
renderHeaderList();
toast('已清空全部请求头','success');
}

function removeProvHeader(key){
delete editingHeaders[key];
renderHeaderList();
}

function renameProvHeader(oldKey,newKey){
newKey=newKey.trim();
if(!newKey||newKey===oldKey)return;
const val=editingHeaders[oldKey];
delete editingHeaders[oldKey];
editingHeaders[newKey]=val;
renderHeaderList();
}

function updateProvHeaderVal(key,val){
editingHeaders[key]=val;
}

function handleHeaderKeydown(e){
if(e.key==='Enter'){e.preventDefault();addProvHeader();}
}

// 按站点+模型测试（真实对话）
async function testModel(providerId,model){
const modal=document.getElementById('modelTestModal');
const content=document.getElementById('modelTestContent');
modal.classList.add('show');
content.innerHTML='<div class="empty"><span class="loading"></span> 正在测试 '+esc(model)+' ...</div>';
try{
const url=API+'/providers/test/'+providerId+'?model='+encodeURIComponent(model);
const res=await fetch(url,{method:'POST'});
const data=await res.json();
// 取站点当前使用的 Key
const prov=allProvidersCache.find(x=>x.id===providerId);
const activeKeys=(prov&&prov.apiKeys||[]).filter(k=>(k.status||'active')==='active');
const testKey=activeKeys.length?activeKeys[0].key:(prov?prov.apiKey:'');
const resultText=data.success?'正常':'异常';
// 组装一键复制文本
let copyStr='站点: '+(prov?prov.name:'')+'\n';
copyStr+='测试 Key: '+(testKey||'-')+'\n';
copyStr+='测试模型: '+model+'\n';
copyStr+='请求地址: '+(data.testUrl||'-')+'\n';
copyStr+='发送消息: '+(data.testMessage||'-')+'\n';
copyStr+='测试结果: '+resultText+' (HTTP '+(data.status||'-')+')\n';
copyStr+=(data.success?('AI 回复: '+(data.content||'')):('错误信息: '+(data.error||'')));
let html='<div style="display:flex;align-items:center;justify-content:space-between;margin-bottom:12px">';
html+='<div style="font-size:15px;font-weight:600">测试结果: <span class="badge badge-'+(data.success?'active':'disabled')+'">'+resultText+'</span></div>';
html+='<button class="copy-btn" onclick="copyText('+JSON.stringify(copyStr)+',this)">📋 一键复制</button>';
html+='</div>';
html+='<div style="margin-bottom:12px">';
html+='<div style="font-size:13px;color:var(--muted);margin-bottom:4px">测试站点</div>';
html+='<div class="mono" style="margin-bottom:8px">'+esc(prov?prov.name:'-')+'</div>';
html+='<div style="font-size:13px;color:var(--muted);margin-bottom:4px">测试 Key</div>';
html+='<div class="mono" style="margin-bottom:8px;word-break:break-all">'+esc(testKey||'-')+'</div>';
html+='<div style="font-size:13px;color:var(--muted);margin-bottom:4px">测试模型</div>';
html+='<div class="mono" style="margin-bottom:8px">'+esc(model)+'</div>';
html+='<div style="font-size:13px;color:var(--muted);margin-bottom:4px">请求地址</div>';
html+='<div class="mono" style="margin-bottom:8px;font-size:12px;word-break:break-all">'+esc(data.testUrl||'-')+'</div>';
html+='<div style="font-size:13px;color:var(--muted);margin-bottom:4px">发送消息</div>';
html+='<div class="mono" style="margin-bottom:8px;padding:8px;background:var(--bg);border-radius:4px">'+esc(data.testMessage||'-')+'</div>';
if(data.reqHeaders){
html+='<div style="font-size:13px;color:var(--muted);margin-bottom:4px">请求头</div>';
html+='<div class="mono" style="margin-bottom:8px;font-size:12px">';
for(const[k,v]of Object.entries(data.reqHeaders)){
if(k==='Authorization'||k==='x-api-key'){html+=esc(k)+': ***<br>';continue;}
html+=esc(k)+': '+esc(v)+'<br>';
}
html+='</div>';
}
html+='</div>';
if(data.success){
html+='<div style="font-size:13px;color:var(--muted);margin-bottom:4px">AI 回复</div>';
html+='<div class="test-success" style="padding:12px;margin-bottom:8px">'+esc(data.content||'(空)')+'</div>';
}else{
html+='<div style="font-size:13px;color:var(--muted);margin-bottom:4px">错误信息</div>';
html+='<div class="test-error" style="padding:12px;margin-bottom:8px">'+esc(data.error||'未知错误')+'</div>';
}
if(data.raw){
html+='<details style="margin-top:8px"><summary style="cursor:pointer;font-size:12px;color:var(--muted)">原始响应</summary><pre class="mono" style="font-size:11px;padding:8px;background:var(--bg);border-radius:4px;overflow-x:hidden;white-space:pre-wrap;word-break:break-word;overflow-wrap:anywhere;margin-top:4px">'+esc(data.raw)+'</pre></details>';
}
content.innerHTML=html;
}catch(e){
content.innerHTML='<div class="test-error">请求失败: '+esc(e.message)+'</div>';
}
}

// --- 中转站导出/导入 ---
function exportProvider(id){
window.location.href=API+'/providers/export/'+id;
toast('正在下载配置文件','success');
}

function importProvider(){
document.getElementById('importInput').value='';
document.getElementById('importModal').classList.add('show');
}

function showOpenCodeModal(){
document.getElementById('opencodeInput').value='';
document.getElementById('opencodeModal').classList.add('show');
}

async function submitOpenCode(){
const text=document.getElementById('opencodeInput').value.trim();
if(!text){toast('请粘贴配置内容','error');return;}
let parsed;
try{
parsed=JSON.parse(text);
}catch(e){
toast('JSON格式错误: '+e.message,'error');
return;
}
const res=await fetch(API+'/providers/import/opencode',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(parsed)});
if(res.ok){
const p=await res.json();
closeModal('opencodeModal');
toast('导入成功: '+(p.name||'')+' ('+(p.models||[]).length+' 个模型)','success');
loadProviders();
}else{
const err=await res.json().catch(()=>({}));
toast('导入失败: '+(err.error||''),'error');
}
}

async function handleImportFile(event){
const file=event.target.files[0];
if(!file)return;
const text=await file.text();
document.getElementById('importInput').value=text;
event.target.value='';
}

async function handleOpenCodeFile(event){
const file=event.target.files[0];
if(!file)return;
const text=await file.text();
document.getElementById('opencodeInput').value=text;
event.target.value='';
}

async function submitImport(){
const text=document.getElementById('importInput').value.trim();
if(!text){toast('请粘贴配置内容','error');return;
}
let parsed;
try{
parsed=JSON.parse(text);
}catch(e){
toast('JSON格式错误: '+e.message,'error');
return;
}
// 判断格式：newapi 连接配置
if(parsed._type==='newapi_channel_conn'){
const res=await fetch(API+'/providers/import/conn',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(parsed)});
if(res.ok){
const p=await res.json();
closeModal('importModal');
toast('导入成功: '+p.name+' (请返回编辑页面一键获取模型)','success');
loadProviders();
}else{
const err=await res.json().catch(()=>({}));
toast('导入失败: '+(err.error||''),'error');
}
return;
}
// 标准中转站配置
const res=await fetch(API+'/providers/import',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(parsed)});
if(res.ok){
closeModal('importModal');
toast('导入成功','success');
loadProviders();
}else{
const err=await res.json().catch(()=>({}));
toast('导入失败: '+(err.error||''),'error');
}
}

// --- URL 一键复制 ---
async function copyText(text,btn){
try{
await navigator.clipboard.writeText(text);
if(btn){
btn.classList.add('copied');
const old=btn.innerHTML;
btn.innerHTML='✓ 已复制';
setTimeout(()=>{btn.classList.remove('copied');btn.innerHTML=old;},1500);
}
toast('已复制到剪贴板','success');
}catch(e){
toast('复制失败','error');
}
}

// --- 模型路由别名 ---
async function loadAliases(){
// 加载全局路由信息
await loadGlobalRoute();
const res=await fetch(API+'/aliases');
const data=await res.json();
const el=document.getElementById('aliasList');
if(!data||data.length===0){
el.innerHTML='<div class="empty">暂无别名，点击右上角添加</div>';
return;
}
const base=location.protocol+'//'+location.host;
let html='<table><tr><th>别名</th><th>目标站点</th><th>实际模型</th><th>调用URL</th><th>操作</th></tr>';
for(const a of data){
html+='<tr>';
html+='<td class="mono">'+esc(a.name)+'</td>';
html+='<td>'+esc(a.providerName||a.providerId)+'</td>';
html+='<td class="mono">'+esc(a.model)+'</td>';
html+='<td><div class="url-with-copy"><span class="mono" style="font-size:11px">'+base+'/v1/chat/completions</span> <button class="copy-btn" onclick="copyText(\''+base+'/v1/chat/completions\',this)">复制</button></div></td>';
html+='<td><button class="btn btn-sm btn-outline" onclick="editAlias(\''+a.id+'\')">编辑</button> <button class="btn btn-sm btn-danger" onclick="deleteAlias(\''+a.id+'\')">删除</button></td>';
html+='</tr>';
}
html+='</table>';
el.innerHTML=html;
}

async function loadGlobalRoute(){
const[sRes,pRes]=await Promise.all([fetch(API+'/settings'),fetch(API+'/providers')]);
const settings=await sRes.json();
const providers=await pRes.json();
allProvidersCache=providers||[];
const el=document.getElementById('globalRouteInfo');
const activeProvider=(providers||[]).find(p=>p.id===settings.activeProviderId);
if(!activeProvider){
el.innerHTML='<div class="empty">未启用任何站点，请在中转站列表点击「启用」</div>';
return;
}
const base=location.protocol+'//'+location.host;
const baseUrl=base+'/v1';
const openaiUrl=base+'/v1/chat/completions';
const anthropicUrl=base+'/v1/messages';
const providerUrl=base+'/v1/chat/completions/p/'+activeProvider.id;
let html='<table>';
html+='<tr><td style="width:120px;color:var(--muted)">启用站点</td><td><strong>'+esc(activeProvider.name)+'</strong> <span class="badge badge-'+activeProvider.format+'">'+activeProvider.format+'</span></td></tr>';
html+='<tr><td style="color:var(--muted)">默认模型</td><td class="mono">'+esc(activeProvider.defaultModel||'未设置')+'</td></tr>';
html+='<tr><td style="color:var(--muted)">站点地址</td><td class="mono" style="font-size:12px">'+esc(activeProvider.baseUrl)+'</td></tr>';
html+='<tr><td style="color:var(--muted)">已启用模型</td><td style="font-size:12px">'+activeProvider.models.filter(m=>!(activeProvider.disabledModels||[]).includes(m)).length+' 个</td></tr>';
html+='<tr><td style="color:var(--muted)">Base URL</td><td><div class="url-with-copy"><span class="mono" style="font-size:11px">'+baseUrl+'</span> <button class="copy-btn" onclick="copyText(\''+baseUrl+'\',this)">复制</button></div></td></tr>';
html+='<tr><td style="color:var(--muted)">OpenAI 调用</td><td><div class="url-with-copy"><span class="mono" style="font-size:11px">'+openaiUrl+'</span> <button class="copy-btn" onclick="copyText(\''+openaiUrl+'\',this)">复制</button></div></td></tr>';
html+='<tr><td style="color:var(--muted)">Anthropic 调用</td><td><div class="url-with-copy"><span class="mono" style="font-size:11px">'+anthropicUrl+'</span> <button class="copy-btn" onclick="copyText(\''+anthropicUrl+'\',this)">复制</button></div></td></tr>';
html+='<tr><td style="color:var(--muted)">指定站点调用</td><td><div class="url-with-copy"><span class="mono" style="font-size:11px">'+providerUrl+'</span> <button class="copy-btn" onclick="copyText(\''+providerUrl+'\',this)">复制</button></div></td></tr>';
html+='</table>';
el.innerHTML=html;
}

function showAliasModal(){
document.getElementById('aliasModalTitle').textContent='添加路由别名';
document.getElementById('aliasId').value='';
document.getElementById('aliasName').value='';
// 填充站点下拉
const sel=document.getElementById('aliasProvider');
sel.innerHTML='';
for(const p of allProvidersCache){
if(p.status==='active')sel.innerHTML+='<option value="'+p.id+'">'+esc(p.name)+' ('+p.format+')</option>';
}
updateAliasModels();
document.getElementById('aliasModal').classList.add('show');
}

// 根据选中的站点填充模型下拉
function updateAliasModels(selectedModel){
const providerId=document.getElementById('aliasProvider').value;
const p=allProvidersCache.find(x=>x.id===providerId);
const modelSel=document.getElementById('aliasModel');
modelSel.innerHTML='';
if(!p)return;
const disabledSet=new Set(p.disabledModels||[]);
for(const m of (p.models||[])){
if(disabledSet.has(m))continue;
const sel2=(m===selectedModel)?' selected':'';
modelSel.innerHTML+='<option value="'+esc(m)+'"'+sel2+'>'+esc(m)+'</option>';
}
}

async function editAlias(id){
const res=await fetch(API+'/aliases');
const data=await res.json();
const a=data.find(x=>x.id===id);
if(!a)return;
document.getElementById('aliasModalTitle').textContent='编辑路由别名';
document.getElementById('aliasId').value=a.id;
document.getElementById('aliasName').value=a.name;
const sel=document.getElementById('aliasProvider');
sel.innerHTML='';
for(const p of allProvidersCache){
const sel2=p.id===a.providerId?' selected':'';
sel.innerHTML+='<option value="'+p.id+'"'+sel2+'>'+esc(p.name)+' ('+p.format+')</option>';
}
updateAliasModels(a.model);
document.getElementById('aliasModal').classList.add('show');
}

async function saveAlias(){
const id=document.getElementById('aliasId').value;
const body={
name:document.getElementById('aliasName').value,
providerId:document.getElementById('aliasProvider').value,
model:document.getElementById('aliasModel').value
};
if(!body.name||!body.providerId||!body.model){
toast('请填写所有字段','error');return;
}
const method=id?'PUT':'POST';
const url=id?API+'/aliases/'+id:API+'/aliases';
const res=await fetch(url,{method,headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});
if(res.ok){
closeModal('aliasModal');
toast(id?'已更新':'已添加','success');
loadAliases();
}else{
toast('操作失败','error');
}
}

async function deleteAlias(id){
if(!confirm('确定删除此别名？'))return;
const res=await fetch(API+'/aliases/'+id,{method:'DELETE'});
if(res.ok){toast('已删除','success');loadAliases();}
}

// --- Keys ---
async function loadKeys(){
const res=await fetch(API+'/keys');
const data=await res.json();
const el=document.getElementById('keyList');
if(!data||data.length===0){
el.innerHTML='<div class="empty">暂无 API Key，点击右上角生成</div>';
return;
}
let html='<table><tr><th>名称</th><th>Key</th><th>创建时间</th><th>状态</th><th>操作</th></tr>';
for(const k of data){
html+='<tr>';
html+='<td>'+esc(k.name||'-')+'</td>';
html+='<td class="mono">'+esc(k.key)+'</td>';
html+='<td>'+fmtDate(k.createdAt)+'</td>';
html+='<td><span class="badge badge-'+k.status+'">'+k.status+'</span></td>';
html+='<td><button class="btn btn-sm btn-outline" onclick="copyKey(\''+esc(k.key)+'\')">复制</button> <button class="btn btn-sm btn-danger" onclick="deleteKey(\''+k.id+'\')">删除</button></td>';
html+='</tr>';
}
html+='</table>';
el.innerHTML=html;
}

function showKeyModal(){
document.getElementById('keyName').value='';
document.getElementById('keyModal').classList.add('show');
}

async function createKey(){
const name=document.getElementById('keyName').value;
const res=await fetch(API+'/keys',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify({name})});
if(res.ok){
const data=await res.json();
closeModal('keyModal');
toast('已生成: '+data.key,'success');
loadKeys();
}
}

async function deleteKey(id){
if(!confirm('确定删除此 Key？'))return;
const res=await fetch(API+'/keys/'+id,{method:'DELETE'});
if(res.ok){toast('已删除','success');loadKeys();}
}

function copyKey(key){
navigator.clipboard.writeText(key);
toast('已复制到剪贴板','success');
}

// --- Stats ---
let statsCurrentPage=1;
const statsPageSize=30;

async function loadStats(){
const sRes=await fetch(API+'/stats');
const stats=await sRes.json();
await loadStatsLogs(statsCurrentPage);
renderStatsOverview(stats);
renderStatsByProvider(stats);
renderStatsByModel(stats);
renderCharts(stats);
}

async function loadStatsLogs(page){
statsCurrentPage=page;
const lRes=await fetch(API+'/logs?page='+page+'&pageSize='+statsPageSize);
const data=await lRes.json();
const logs=data.logs||[];
let lg='<table><tr><th>时间</th><th>站点</th><th>模型</th><th>格式</th><th>输入</th><th>输出</th></tr>';
for(const l of logs){
lg+='<tr><td>'+fmtDate(l.timestamp)+'</td><td>'+esc(l.providerName)+'</td><td class="mono">'+esc(l.model)+'</td><td>'+l.clientFormat+'</td><td>'+l.inputTokens+'</td><td>'+l.outputTokens+'</td></tr>';
}
if(logs.length===0)lg+='<tr><td colspan="6" class="empty">暂无日志</td></tr>';
lg+='</table>';
document.getElementById('recentLogs').innerHTML=lg;
// 分页
renderLogPagination(data.page||1,data.pages||0,data.total||0);
}

function renderStatsOverview(stats){
document.getElementById('statsGrid').innerHTML=
statCard(stats.totalReqs||0,'总请求数')+
statCard((stats.totalInput||0).toLocaleString(),'输入 Token')+
statCard((stats.totalOutput||0).toLocaleString(),'输出 Token')+
statCard((stats.totalTokens||0).toLocaleString(),'Token 总量');
}

function renderStatsByProvider(stats){
let bp='<table><tr><th>站点</th><th>请求数</th><th>输入Token</th><th>输出Token</th><th>总量</th></tr>';
const providers=stats.byProvider||{};
for(const[name,v]of Object.entries(providers)){
bp+='<tr><td>'+esc(name)+'</td><td>'+v.count+'</td><td>'+v.input+'</td><td>'+v.output+'</td><td>'+(v.total||0).toLocaleString()+'</td></tr>';
}
if(Object.keys(providers).length===0)bp+='<tr><td colspan="5" class="empty">暂无数据</td></tr>';
bp+='</table>';
document.getElementById('statsByProvider').innerHTML=bp;
}

function renderStatsByModel(stats){
let bm='<table><tr><th>模型</th><th>请求数</th><th>输入Token</th><th>输出Token</th><th>总量</th></tr>';
const models=stats.byModel||{};
for(const[name,v]of Object.entries(models)){
bm+='<tr><td class="mono">'+esc(name)+'</td><td>'+v.count+'</td><td>'+v.input+'</td><td>'+v.output+'</td><td>'+(v.total||0).toLocaleString()+'</td></tr>';
}
if(Object.keys(models).length===0)bm+='<tr><td colspan="5" class="empty">暂无数据</td></tr>';
bm+='</table>';
document.getElementById('statsByModel').innerHTML=bm;
}

function renderLogPagination(page,pages,total){
const el=document.getElementById('logPagination');
if(pages<=1){el.innerHTML='<span style="color:var(--muted);font-size:12px">共 '+total+' 条</span>';return;}
let html='';
html+='<button class="btn btn-sm btn-outline" '+(page<=1?'disabled':'')+' onclick="loadStatsLogs('+(page-1)+')">上一页</button>';
html+='<span style="font-size:13px">第 '+page+'/'+pages+' 页 (共'+total+'条)</span>';
html+='<button class="btn btn-sm btn-outline" '+(page>=pages?'disabled':'')+' onclick="loadStatsLogs('+(page+1)+')">下一页</button>';
el.innerHTML=html;
}

async function clearLogs(){
if(!confirm('确定清空所有请求日志？此操作不可恢复。'))return;
const res=await fetch(API+'/logs/clear',{method:'POST'});
if(res.ok){
toast('已清空','success');
statsCurrentPage=1;
loadStats();
}else{
toast('清空失败','error');
}
}

function statCard(val,label){
return '<div class="stat-card"><div class="stat-value">'+val+'</div><div class="stat-label">'+label+'</div></div>';
}

let chartDailyInst=null;
let chartProviderInst=null;
let chartModelInst=null;

function renderCharts(stats){
const dailyData=stats.byDate||{};
const providerData=stats.byProvider||{};
const modelData=stats.byModel||{};
// 日期趋势图
const dailyLabels=Object.keys(dailyData).sort();
const dailyInput=dailyLabels.map(d=>dailyData[d].input||0);
const dailyOutput=dailyLabels.map(d=>dailyData[d].output||0);
const dailyReqs=dailyLabels.map(d=>dailyData[d].count||0);
if(chartDailyInst)chartDailyInst.destroy();
const ctxDaily=document.getElementById('chartDaily').getContext('2d');
chartDailyInst=new Chart(ctxDaily,{type:'line',data:{labels:dailyLabels,datasets:[{label:'输入 Token',data:dailyInput,borderColor:'#3b82f6',backgroundColor:'rgba(59,130,246,.1)',fill:true,tension:.3},{label:'输出 Token',data:dailyOutput,borderColor:'#10b981',backgroundColor:'rgba(16,185,129,.1)',fill:true,tension:.3}]},options:{responsive:true,maintainAspectRatio:false,interaction:{mode:'index',intersect:false},plugins:{legend:{position:'top'}},scales:{y:{beginAtZero:true}}}});
// 中转站柱状图
const provLabels=Object.keys(providerData);
const provReqs=provLabels.map(p=>providerData[p].count||0);
const provTokens=provLabels.map(p=>providerData[p].total||0);
if(chartProviderInst)chartProviderInst.destroy();
const ctxProv=document.getElementById('chartProvider').getContext('2d');
chartProviderInst=new Chart(ctxProv,{type:'bar',data:{labels:provLabels,datasets:[{label:'请求数',data:provReqs,backgroundColor:'rgba(59,130,246,.7)',order:2},{label:'Token 总量',data:provTokens,backgroundColor:'rgba(16,185,129,.7)',order:1}]},options:{responsive:true,maintainAspectRatio:false,plugins:{legend:{position:'top'}},scales:{y:{beginAtZero:true}}}});
// 模型饼图
const modelLabels=Object.keys(modelData);
const modelTokens=modelLabels.map(m=>modelData[m].total||0);
const modelColors=['#3b82f6','#10b981','#f59e0b','#ef4444','#8b5cf6','#ec4899','#06b6d4','#84cc16','#f97316','#6366f1'];
if(chartModelInst)chartModelInst.destroy();
const ctxModel=document.getElementById('chartModel').getContext('2d');
chartModelInst=new Chart(ctxModel,{type:'doughnut',data:{labels:modelLabels,datasets:[{data:modelTokens,backgroundColor:modelColors.slice(0,modelLabels.length)}]},options:{responsive:true,maintainAspectRatio:false,plugins:{legend:{position:'right'}}}});
}

// --- Settings ---
async function loadSettings(){
const res=await fetch(API+'/settings');
const data=await res.json();
activeProviderId=data.activeProviderId||'';
if(data.inputPresets&&data.inputPresets.length)inputPresets=data.inputPresets;
if(data.outputPresets&&data.outputPresets.length)outputPresets=data.outputPresets;
renderPresetEditors();
loadGlobalModelConfigs();
}

// 填充预设下拉框
function fillPresetSelect(selId,selectedVal,presets){
const sel=document.getElementById(selId);
sel.innerHTML='<option value="">不限制(走客户端)</option>';
for(const p of presets){
const s=(p===selectedVal)?' selected':'';
sel.innerHTML+='<option value="'+esc(p)+'"'+s+'>'+esc(p)+'</option>';
}
}

// 渲染设置页的预设标签
function renderPresetEditors(){
renderPresetTags('inputPresetsWrap','inputPresetInput',inputPresets);
renderPresetTags('outputPresetsWrap','outputPresetInput',outputPresets);
}

function renderPresetTags(wrapId,inputId,arr){
const wrap=document.getElementById(wrapId);
const input=document.getElementById(inputId);
// 清除旧 tag（保留 input）
const tags=wrap.querySelectorAll('.tag');
tags.forEach(t=>t.remove());
arr.forEach((v,i)=>{
const tag=document.createElement('span');
tag.className='tag';
tag.innerHTML=esc(v)+'<button class="tag-x" onclick="removePresetTag(\''+wrapId+'\','+i+',\''+inputId+'\')">×</button>';
wrap.insertBefore(tag,input);
});
}

function removePresetTag(wrapId,index,inputId){
const isInput=wrapId==='inputPresetsWrap';
if(isInput)inputPresets.splice(index,1);
else outputPresets.splice(index,1);
renderPresetEditors();
}

function handlePresetKeydown(e,type){
if(e.key==='Enter'||e.key===','){
e.preventDefault();
const inputId=type==='input'?'inputPresetInput':'outputPresetInput';
const val=document.getElementById(inputId).value.trim().replace(/,$/,'');
if(val){
if(type==='input'&&!inputPresets.includes(val))inputPresets.push(val);
if(type==='output'&&!outputPresets.includes(val))outputPresets.push(val);
document.getElementById(inputId).value='';
renderPresetEditors();
}
}
}

async function saveSettings(){
// 合并输入框中未提交的内容
const ipVal=document.getElementById('inputPresetInput').value.trim();
if(ipVal&&!inputPresets.includes(ipVal))inputPresets.push(ipVal);
const opVal=document.getElementById('outputPresetInput').value.trim();
if(opVal&&!outputPresets.includes(opVal))outputPresets.push(opVal);
const res=await fetch(API+'/settings',{method:'PUT',headers:{'Content-Type':'application/json'},body:JSON.stringify({activeProviderId:activeProviderId,inputPresets:inputPresets,outputPresets:outputPresets})});
if(res.ok){
toast('已保存','success');
}
}

// --- 全局模型上下文配置 ---
async function loadGlobalModelConfigs(){
const res=await fetch(API+'/providers');
const providers=await res.json();
allProvidersCache=providers||[];
const el=document.getElementById('globalModelConfigList');
// 列出所有站点+模型的配置（站点+模型为唯一键）
let rows=[];
for(const p of providers||[]){
for(const mc of(p.modelConfigs||[])){
rows.push({providerId:p.id,providerName:p.name,model:mc.model,inputLimit:mc.inputLimit,outputLimit:mc.outputLimit});
}
}
if(rows.length===0){
el.innerHTML='<div class="empty">暂无配置，点击右上角添加</div>';
return;
}
let html='<table><tr><th>站点</th><th>模型</th><th>输入预算</th><th>输出预算</th><th>操作</th></tr>';
for(const r of rows){
html+='<tr>';
html+='<td>'+esc(r.providerName)+'</td>';
html+='<td class="mono">'+esc(r.model)+'</td>';
html+='<td>'+(r.inputLimit||'-')+'</td>';
html+='<td>'+(r.outputLimit||'-')+'</td>';
html+='<td><button class="btn btn-sm btn-outline" onclick="editGlobalModelConfig(\''+r.providerId+'\',\''+esc(r.model)+'\')">编辑</button> <button class="btn btn-sm btn-danger" onclick="deleteGlobalModelConfig(\''+r.providerId+'\',\''+esc(r.model)+'\')">删除</button></td>';
html+='</tr>';
}
html+='</table>';
el.innerHTML=html;
}

function showGlobalModelConfigModal(){
document.getElementById('globalMcTitle').textContent='添加模型上下文配置';
// 填充站点下拉
const provSel=document.getElementById('gmcProvider');
provSel.innerHTML='';
for(const p of allProvidersCache){
provSel.innerHTML+='<option value="'+p.id+'">'+esc(p.name)+' ('+p.format+')</option>';
}
onGmcProviderChange();
fillPresetSelect('gmcInputLimit','',inputPresets);
fillPresetSelect('gmcOutputLimit','',outputPresets);
document.getElementById('globalModelConfigModal').classList.add('show');
}

function onGmcProviderChange(){
const pid=document.getElementById('gmcProvider').value;
const p=allProvidersCache.find(x=>x.id===pid);
const modelSel=document.getElementById('gmcModel');
modelSel.innerHTML='';
if(!p)return;
const disabledSet=new Set(p.disabledModels||[]);
for(const m of(p.models||[])){
if(!disabledSet.has(m))modelSel.innerHTML+='<option value="'+esc(m)+'">'+esc(m)+'</option>';
}
}

function editGlobalModelConfig(providerId,model){
document.getElementById('globalMcTitle').textContent='编辑模型上下文配置';
// 填充站点下拉并选中
const provSel=document.getElementById('gmcProvider');
provSel.innerHTML='';
for(const p of allProvidersCache){
const s=(p.id===providerId)?' selected':'';
provSel.innerHTML+='<option value="'+p.id+'"'+s+'>'+esc(p.name)+' ('+p.format+')</option>';
}
onGmcProviderChange();
// 选中模型
const modelSel=document.getElementById('gmcModel');
for(const opt of modelSel.options){
if(opt.value===model){opt.selected=true;break;}
}
// 填充已有配置
const p=allProvidersCache.find(x=>x.id===providerId);
const mc=p?(p.modelConfigs||[]).find(c=>c.model===model):null;
fillPresetSelect('gmcInputLimit',mc?mc.inputLimit:'',inputPresets);
fillPresetSelect('gmcOutputLimit',mc?mc.outputLimit:'',outputPresets);
document.getElementById('globalModelConfigModal').classList.add('show');
}

async function saveGlobalModelConfig(){
const providerId=document.getElementById('gmcProvider').value;
const model=document.getElementById('gmcModel').value;
if(!providerId||!model){toast('请选择站点和模型','error');return;}
const inputLimit=document.getElementById('gmcInputLimit').value;
const outputLimit=document.getElementById('gmcOutputLimit').value;
const body={model:model,inputLimit:inputLimit,outputLimit:outputLimit};
const url=API+'/providers/models/config/'+providerId+'?model='+encodeURIComponent(model);
const res=await fetch(url,{method:'PUT',headers:{'Content-Type':'application/json'},body:JSON.stringify(body)});
if(res.ok){
closeModal('globalModelConfigModal');
toast('已保存','success');
loadGlobalModelConfigs();
}else{
toast('保存失败','error');
}
}

async function deleteGlobalModelConfig(providerId,model){
if(!confirm('确定删除 '+model+' 的上下文配置？'))return;
const url=API+'/providers/models/config/'+providerId+'?model='+encodeURIComponent(model);
const res=await fetch(url,{method:'PUT',headers:{'Content-Type':'application/json'},body:JSON.stringify({model:model,inputLimit:'',outputLimit:''})});
if(res.ok){
toast('已删除','success');
loadGlobalModelConfigs();
}else{
toast('删除失败','error');
}
}

// --- Config ---
function loadConfig(){
const port=location.port||'3458';
const base='http://'+location.hostname+':'+port;
document.getElementById('cfgOpenAIUrl').textContent=base+'/v1';
document.getElementById('cfgAnthropicUrl').textContent=base;
document.getElementById('cfgHealthUrl').textContent=base+'/health';
document.getElementById('cfgPOpenAI').textContent=base+'/v1/chat/completions/p/{站点ID}';
document.getElementById('cfgPAnthropic').textContent=base+'/v1/messages/p/{站点ID}';
}

// --- Utils ---
function closeModal(id){document.getElementById(id).classList.remove('show');}
function esc(s){return String(s||'').replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;').replace(/'/g,'&#39;');}
function fmtDate(s){if(!s)return'-';const d=new Date(s);return d.toLocaleString('zh-CN',{month:'2-digit',day:'2-digit',hour:'2-digit',minute:'2-digit'});}
let toastTimer=null;
function toast(msg,type){
const el=document.getElementById('toast');
el.className='toast toast-'+type;
el.textContent=msg;
el.style.display='block';
if(toastTimer)clearTimeout(toastTimer);
toastTimer=setTimeout(()=>{el.style.display='none';el.className='';},3000);
}

// 弹窗只能通过取消/保存按钮关闭，点击遮罩不关闭

initTheme();
loadProviders();
loadKeys();
</script>
<script src="/static/chart.umd.min.js"></script>
</body>
</html>`
