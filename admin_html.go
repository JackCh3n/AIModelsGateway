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
.container{max-width:1100px;margin:0 auto;padding:24px}
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
.test-result{margin-top:12px;padding:12px;border-radius:var(--radius);font-size:13px}
.test-success{background:rgba(16,185,129,.08);border:1px solid rgba(16,185,129,.25)}
.test-error{background:rgba(239,68,68,.08);border:1px solid rgba(239,68,68,.25)}
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
.model-chip{display:inline-flex;align-items:center;gap:6px;background:var(--tag-bg);color:var(--tag-text);padding:3px 8px;border-radius:4px;font-size:12px;font-family:'Courier New',monospace;margin:2px}
.model-chip.disabled{opacity:.45;text-decoration:line-through}
.model-chip .chip-toggle{cursor:pointer;border:none;background:none;color:inherit;font-size:12px;padding:0}
.model-chip .chip-toggle:hover{opacity:.8}
.model-row{margin-top:8px}
.url-hint{font-size:11px;color:var(--muted);font-family:'Courier New',monospace;margin-top:4px;word-break:break-all}
.expand-btn{font-size:12px;color:var(--accent);cursor:pointer;background:none;border:none;padding:2px 6px}
.copy-btn{display:inline-flex;align-items:center;gap:4px;padding:2px 8px;font-size:11px;border-radius:4px;border:1px solid var(--border);background:var(--card);color:var(--muted);cursor:pointer;transition:.2s;vertical-align:middle}
.copy-btn:hover{border-color:var(--accent);color:var(--accent)}
.copy-btn.copied{background:var(--green);color:#fff;border-color:var(--green)}
.url-with-copy{display:flex;align-items:center;gap:6px;flex-wrap:wrap}
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
<div class="info" id="headerInfo">加载中...</div>
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
<button class="tab" onclick="switchTab(event,'settings')">设置</button>
<button class="tab" onclick="switchTab(event,'config')">接入配置</button>
</div>

<!-- 中转站管理 -->
<div class="panel active" id="panel-providers">
<div class="card">
<div class="card-title">中转站列表 <div style="display:flex;gap:8px"><button class="btn btn-outline" onclick="importProvider()">导入配置</button><button class="btn btn-primary" onclick="showProviderModal()">+ 添加中转站</button></div></div>
<div id="providerList"></div>
<input type="file" id="importFile" accept=".json" style="display:none" onchange="handleImportFile(event)">
</div>
</div>

<!-- 模型路由 -->
<div class="panel" id="panel-aliases">
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
<div class="card-title">按中转站统计</div>
<div id="statsByProvider"></div>
</div>
<div class="card">
<div class="card-title">按模型统计</div>
<div id="statsByModel"></div>
</div>
<div class="card">
<div class="card-title">最近请求日志</div>
<div id="recentLogs"></div>
</div>
</div>

<!-- 设置 -->
<div class="panel" id="panel-settings">
<div class="card">
<div class="card-title">全局设置</div>
<div class="form-group">
<label>默认模型</label>
<input class="input" id="settingDefaultModel" placeholder="gpt-4o-mini">
</div>
<button class="btn btn-primary" onclick="saveSettings()">保存设置</button>
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
<div class="config-line">Model: <span id="cfgModel">gpt-4o-mini</span></div>
</div>
<div class="config-box">
<h4>Anthropic 格式 (Claude Code / Cline)</h4>
<div class="config-line">Base URL: <span id="cfgAnthropicUrl">http://127.0.0.1:3458</span> <button class="copy-btn" onclick="copyText(document.getElementById('cfgAnthropicUrl').textContent,this)">复制</button></div>
<div class="config-line">API Key: <span id="cfgAnthropicKey">在 API Keys 页面生成</span></div>
<div class="config-line">Model: <span id="cfgModel2">gpt-4o-mini</span></div>
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
<label>API Keys (多Key轮询)</label>
<div id="provKeysList" style="margin-bottom:8px"></div>
<div style="display:flex;gap:8px">
<input class="input" id="provKeyInput" placeholder="sk-..." style="flex:1" oninput="this.value=this.value.trim()" onkeydown="handleKeyKeydown(event)">
<input class="input" id="provKeyName" placeholder="备注(可选)" style="width:120px">
<button class="btn btn-outline" onclick="addProvKey()">添加</button>
</div>
</div>
<div class="form-group">
<label>支持的模型 (回车添加，或逗号分隔添加多个)</label>
<div class="tag-input-wrap" id="provModelsWrap">
<input class="tag-input" id="provModelInput" placeholder="输入模型名后回车..." oninput="this.value=this.value.trim()" onkeydown="handleModelKeydown(event)">
</div>
</div>
<div class="form-group">
<label>自定义请求头 (每行一个，格式: Key: Value)</label>
<textarea class="input" id="provHeaders" rows="3" placeholder="X-DashScope-Wait-Timeout: 30&#10;X-Custom-Header: value"></textarea>
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
}

// --- 模型标签输入 ---
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
function renderKeyList(){
const el=document.getElementById('provKeysList');
el.innerHTML='';
if(editingKeys.length===0){
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
let html='<input class="input key-edit-val" value="'+esc(k.key)+'" oninput="this.value=this.value.trim();editingKeys['+i+'].key=this.value" style="flex:1;font-size:12px;padding:4px 8px" title="可编辑">';
html+='<input class="input key-edit-name" value="'+esc(k.name||'')+'" placeholder="备注" oninput="editingKeys['+i+'].name=this.value" style="width:100px;font-size:12px;padding:4px 8px">';
html+='<span class="badge badge-'+(k.status||'active')+'">'+(k.status||'active')+'</span>';
html+='<button class="copy-btn" onclick="toggleProvKey('+i+')" title="'+(isActive?'禁用':'启用')+'">'+(isActive?'⏸️':'▶️')+'</button>';
html+='<button class="copy-btn" onclick="testProvKey('+i+')">测试</button>';
html+='<button class="key-x" onclick="removeProvKey('+i+')" title="删除">×</button>';
item.innerHTML=html;
el.appendChild(item);
});
}
function addProvKey(){
const keyInput=document.getElementById('provKeyInput');
const nameInput=document.getElementById('provKeyName');
const key=keyInput.value.trim();
if(!key){toast('请输入 Key','error');return;}
editingKeys.push({id:'',key:key,name:nameInput.value.trim(),status:'active'});
keyInput.value='';
nameInput.value='';
renderKeyList();
keyInput.focus();
}
function removeProvKey(i){
editingKeys.splice(i,1);
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
customHeaders:parseCustomHeaders(document.getElementById('provHeaders').value)
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
html+='<details style="margin-top:8px"><summary style="cursor:pointer;font-size:12px;color:var(--muted)">原始响应</summary><pre class="mono" style="font-size:11px;padding:8px;background:var(--bg);border-radius:4px;overflow-x:auto;margin-top:4px;white-space:pre-wrap">'+esc(data.raw)+'</pre></details>';
}
content.innerHTML=html;
}catch(e){
content.innerHTML='<div class="test-error">请求失败: '+esc(e.message)+'</div>';
}
}

// --- Providers ---
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

let html='<table><tr><th>名称</th><th>格式</th><th>Base URL</th><th>模型</th><th>Keys</th><th>状态</th><th>操作</th></tr>';
for(const p of data){
const isActive=p.id===activeProviderId;
const disabledSet=new Set(p.disabledModels||[]);
const enabledCount=(p.models||[]).filter(m=>!disabledSet.has(m)).length;
const totalCount=(p.models||[]).length;
const activeKeyCount=(p.apiKeys||[]).filter(k=>k.status==='active').length;
const totalKeyCount=(p.apiKeys||[]).length;
html+='<tr>';
html+='<td>'+esc(p.name)+(isActive?' <span class="badge badge-current">当前</span>':'')+'</td>';
html+='<td><span class="badge badge-'+p.format+'">'+p.format+'</span></td>';
html+='<td class="mono">'+esc(p.baseUrl)+'</td>';
html+='<td>'+enabledCount+'/'+totalCount+' <button class="expand-btn" onclick="toggleModels(\''+p.id+'\')">展开</button></td>';
html+='<td>'+activeKeyCount+'/'+totalKeyCount+' keys</td>';
html+='<td><span class="badge badge-'+p.status+'">'+p.status+'</span></td>';
html+='<td>';
if(!isActive&&p.status==='active')html+='<button class="btn btn-sm btn-success" onclick="setActive(\''+p.id+'\')">启用</button> ';
html+='<button class="btn btn-sm btn-outline" onclick="editProvider(\''+p.id+'\')">编辑</button> ';
html+='<button class="btn btn-sm btn-outline" onclick="exportProvider(\''+p.id+'\')">导出</button> ';
html+='<button class="btn btn-sm btn-danger" onclick="deleteProvider(\''+p.id+'\')">删除</button>';
html+='</td>';
html+='</tr>';
// 模型展开行
html+='<tr id="models-'+p.id+'" style="display:none"><td colspan="7"><div class="model-row">';
for(const m of (p.models||[])){
const enabled=!disabledSet.has(m);
const isDefault=(m===defaultModel);
const modelUrl=getModelUrl(p,m);
html+='<span class="model-chip'+(enabled?'':' disabled')+'">';
html+='<button class="chip-toggle" onclick="toggleModel(\''+p.id+'\',\''+esc(m)+'\')" title="'+(enabled?'点击禁用':'点击启用')+'">'+(enabled?'✅':'⏸️')+'</button> ';
html+=esc(m);
if(isDefault)html+=' <span class="badge badge-current">默认</span>';
html+=' <span class="url-hint" style="display:block">'+modelUrl+' <button class="copy-btn" onclick="copyText(\''+esc(modelUrl)+'\',this)">复制</button> <button class="copy-btn" onclick="testModel(\''+p.id+'\',\''+esc(m)+'\')">测试</button>'+(isDefault?'':' <button class="copy-btn" onclick="setModelDefault(\''+esc(m)+'\')">设为默认</button>')+'</span>';
html+='</span>';
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

async function setModelDefault(model){
const res=await fetch(API+'/settings',{method:'PUT',headers:{'Content-Type':'application/json'},body:JSON.stringify({defaultModel:model,activeProviderId:activeProviderId})});
if(res.ok){
toast('已设为默认模型: '+model,'success');
loadProviders();
}else{
toast('操作失败','error');
}
}

function showProviderModal(){
document.getElementById('providerModalTitle').textContent='添加中转站';
document.getElementById('provId').value='';
document.getElementById('provName').value='';
document.getElementById('provFormat').value='openai';
document.getElementById('provStatus').value='active';
document.getElementById('provBaseUrl').value='';
document.getElementById('provKeyInput').value='';
document.getElementById('provKeyName').value='';
editingKeys=[];
renderKeyList();
document.getElementById('provHeaders').value='';
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
// 加载多 Key
editingKeys=(p.apiKeys||[]).map(k=>({...k}));
renderKeyList();
// 加载自定义请求头
let hdrText='';
if(p.customHeaders){
for(const[k,v]of Object.entries(p.customHeaders)){hdrText+=k+': '+v+'\n';}
}
document.getElementById('provHeaders').value=hdrText.trim();
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
// 解析自定义请求头
const customHeaders=parseCustomHeaders(document.getElementById('provHeaders').value);
const body={
name:document.getElementById('provName').value,
format:document.getElementById('provFormat').value,
status:document.getElementById('provStatus').value,
baseUrl:document.getElementById('provBaseUrl').value,
apiKey:editingKeys.length>0?editingKeys[0].key:'',
apiKeys:editingKeys.map(k=>({id:k.id||'',key:k.key,name:k.name||'',status:k.status||'active'})),
models:[...editingModels],
disabledModels:[],
customHeaders:customHeaders
};
if(!body.name||!body.baseUrl){
toast('请填写名称和 Base URL','error');return;
}
// 编辑时保留 disabledModels
if(id){
const old=allProvidersCache.find(x=>x.id===id);
if(old)body.disabledModels=old.disabledModels||[];
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
customHeaders:parseCustomHeaders(document.getElementById('provHeaders').value)
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

// 解析自定义请求头文本为对象
function parseCustomHeaders(text){
const headers={};
if(!text||!text.trim())return headers;
const lines=text.split('\n');
for(const line of lines){
const trimmed=line.trim();
if(!trimmed)continue;
const idx=trimmed.indexOf(':');
if(idx>0){
const key=trimmed.substring(0,idx).trim();
const val=trimmed.substring(idx+1).trim();
if(key)headers[key]=val;
}
}
return headers;
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
let html='<div style="margin-bottom:12px">';
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
html+='<div class="test-success" style="margin-bottom:8px"><strong>测试成功</strong> (HTTP '+data.status+')</div>';
html+='<div style="font-size:13px;color:var(--muted);margin-bottom:4px">AI 回复</div>';
html+='<div class="test-success" style="padding:12px;margin-bottom:8px">'+esc(data.content||'(空)')+'</div>';
}else{
html+='<div class="test-error" style="margin-bottom:8px"><strong>测试失败</strong> (HTTP '+data.status+')</div>';
html+='<div class="test-error" style="padding:12px;margin-bottom:8px">'+esc(data.error||'未知错误')+'</div>';
}
if(data.raw){
html+='<details style="margin-top:8px"><summary style="cursor:pointer;font-size:12px;color:var(--muted)">原始响应</summary><pre class="mono" style="font-size:11px;padding:8px;background:var(--bg);border-radius:4px;overflow-x:auto;margin-top:4px;white-space:pre-wrap">'+esc(data.raw)+'</pre></details>';
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
document.getElementById('importFile').click();
}

async function handleImportFile(event){
const file=event.target.files[0];
if(!file)return;
const text=await file.text();
try{
const p=JSON.parse(text);
const res=await fetch(API+'/providers/import',{method:'POST',headers:{'Content-Type':'application/json'},body:JSON.stringify(p)});
if(res.ok){
toast('导入成功','success');
loadProviders();
}else{
toast('导入失败','error');
}
}catch(e){
toast('文件格式错误: '+e.message,'error');
}
event.target.value='';
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
async function loadStats(){
const [sRes,lRes]=await Promise.all([fetch(API+'/stats'),fetch(API+'/logs')]);
const stats=await sRes.json();
const logs=await lRes.json();

document.getElementById('statsGrid').innerHTML=
statCard(stats.totalReqs||0,'总请求数')+
statCard((stats.totalInput||0).toLocaleString(),'输入 Token')+
statCard((stats.totalOutput||0).toLocaleString(),'输出 Token')+
statCard((stats.totalTokens||0).toLocaleString(),'Token 总量');

let bp='<table><tr><th>站点</th><th>请求数</th><th>输入Token</th><th>输出Token</th><th>总量</th></tr>';
const providers=stats.byProvider||{};
for(const[name,v]of Object.entries(providers)){
bp+='<tr><td>'+esc(name)+'</td><td>'+v.count+'</td><td>'+v.input+'</td><td>'+v.output+'</td><td>'+(v.total||0).toLocaleString()+'</td></tr>';
}
if(Object.keys(providers).length===0)bp+='<tr><td colspan="5" class="empty">暂无数据</td></tr>';
bp+='</table>';
document.getElementById('statsByProvider').innerHTML=bp;

let bm='<table><tr><th>模型</th><th>请求数</th><th>输入Token</th><th>输出Token</th><th>总量</th></tr>';
const models=stats.byModel||{};
for(const[name,v]of Object.entries(models)){
bm+='<tr><td class="mono">'+esc(name)+'</td><td>'+v.count+'</td><td>'+v.input+'</td><td>'+v.output+'</td><td>'+(v.total||0).toLocaleString()+'</td></tr>';
}
if(Object.keys(models).length===0)bm+='<tr><td colspan="5" class="empty">暂无数据</td></tr>';
bm+='</table>';
document.getElementById('statsByModel').innerHTML=bm;

let lg='<table><tr><th>时间</th><th>站点</th><th>模型</th><th>格式</th><th>输入</th><th>输出</th></tr>';
for(const l of (logs||[]).slice(0,50)){
lg+='<tr><td>'+fmtDate(l.timestamp)+'</td><td>'+esc(l.providerName)+'</td><td class="mono">'+esc(l.model)+'</td><td>'+l.clientFormat+'</td><td>'+l.inputTokens+'</td><td>'+l.outputTokens+'</td></tr>';
}
if(!logs||logs.length===0)lg+='<tr><td colspan="6" class="empty">暂无日志</td></tr>';
lg+='</table>';
document.getElementById('recentLogs').innerHTML=lg;
}

function statCard(val,label){
return '<div class="stat-card"><div class="stat-value">'+val+'</div><div class="stat-label">'+label+'</div></div>';
}

// --- Settings ---
async function loadSettings(){
const res=await fetch(API+'/settings');
const data=await res.json();
document.getElementById('settingDefaultModel').value=data.defaultModel||'gpt-4o-mini';
activeProviderId=data.activeProviderId||'';
}

async function saveSettings(){
const model=document.getElementById('settingDefaultModel').value;
const res=await fetch(API+'/settings',{method:'PUT',headers:{'Content-Type':'application/json'},body:JSON.stringify({defaultModel:model,activeProviderId:activeProviderId})});
if(res.ok)toast('已保存','success');
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

// 弹窗：点击遮罩空白处关闭，但点击弹窗内容不关闭
document.querySelectorAll('.modal-overlay').forEach(o=>{
o.addEventListener('click',e=>{
// 只有直接点击遮罩层本身才关闭，点击子元素(弹窗内容)不关闭
if(e.target===o)o.classList.remove('show');
});
});
// 阻止弹窗内容点击事件冒泡
document.querySelectorAll('.modal').forEach(m=>{
m.addEventListener('click',e=>{e.stopPropagation();});
});

initTheme();
loadProviders();
loadKeys();
</script>
</body>
</html>`
