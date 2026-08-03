package main

const adminHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>AI Models Gateway</title>
<style>
*{margin:0;padding:0;box-sizing:border-box}
:root{--bg:#1a1a2e;--card:#16213e;--border:#0f3460;--accent:#e94560;--text:#eee;--muted:#999;--green:#4ecca3;--blue:#4fc3f7;--radius:8px}
body{font-family:'Segoe UI',system-ui,sans-serif;background:var(--bg);color:var(--text);min-height:100vh}
.header{background:var(--card);border-bottom:2px solid var(--accent);padding:16px 24px;display:flex;align-items:center;justify-content:space-between}
.header h1{font-size:20px;font-weight:600}
.header h1 span{color:var(--accent)}
.header .info{font-size:13px;color:var(--muted)}
.container{max-width:1100px;margin:0 auto;padding:24px}
.tabs{display:flex;gap:4px;margin-bottom:20px}
.tab{padding:10px 20px;background:var(--card);border:none;border-radius:var(--radius) var(--radius) 0 0;cursor:pointer;color:var(--muted);font-size:14px;transition:.2s}
.tab:hover{color:var(--text)}
.tab.active{background:var(--border);color:var(--text)}
.panel{display:none}
.panel.active{display:block}
.card{background:var(--card);border:1px solid var(--border);border-radius:var(--radius);padding:20px;margin-bottom:16px}
.card-title{font-size:16px;font-weight:600;margin-bottom:16px;display:flex;justify-content:space-between;align-items:center}
table{width:100%;border-collapse:collapse;font-size:14px}
th{text-align:left;padding:10px 12px;border-bottom:2px solid var(--border);color:var(--muted);font-weight:500;font-size:12px;text-transform:uppercase}
td{padding:10px 12px;border-bottom:1px solid var(--border)}
tr:hover{background:rgba(255,255,255,.03)}
.badge{display:inline-block;padding:2px 8px;border-radius:4px;font-size:12px;font-weight:500}
.badge-active{background:rgba(78,204,163,.15);color:var(--green)}
.badge-disabled{background:rgba(233,69,96,.15);color:var(--accent)}
.badge-openai{background:rgba(79,195,247,.15);color:var(--blue)}
.badge-anthropic{background:rgba(233,69,96,.15);color:var(--accent)}
.badge-current{background:var(--accent);color:#fff}
.btn{padding:8px 16px;border:none;border-radius:var(--radius);cursor:pointer;font-size:13px;font-weight:500;transition:.2s}
.btn-primary{background:var(--accent);color:#fff}
.btn-primary:hover{opacity:.85}
.btn-sm{padding:4px 10px;font-size:12px}
.btn-danger{background:rgba(233,69,96,.2);color:var(--accent);border:1px solid rgba(233,69,96,.3)}
.btn-danger:hover{background:rgba(233,69,96,.3)}
.btn-success{background:rgba(78,204,163,.2);color:var(--green);border:1px solid rgba(78,204,163,.3)}
.btn-success:hover{background:rgba(78,204,163,.3)}
.btn-outline{background:transparent;border:1px solid var(--border);color:var(--text)}
.btn-outline:hover{border-color:var(--accent)}
.input,.select,textarea{width:100%;padding:8px 12px;background:var(--bg);border:1px solid var(--border);border-radius:var(--radius);color:var(--text);font-size:14px;font-family:inherit}
.input:focus,.select:focus,textarea:focus{outline:none;border-color:var(--accent)}
.form-group{margin-bottom:14px}
.form-group label{display:block;margin-bottom:6px;font-size:13px;color:var(--muted)}
.form-row{display:flex;gap:12px}
.form-row .form-group{flex:1}
.modal-overlay{position:fixed;inset:0;background:rgba(0,0,0,.6);display:none;align-items:center;justify-content:center;z-index:100}
.modal-overlay.show{display:flex}
.modal{background:var(--card);border:1px solid var(--border);border-radius:var(--radius);padding:24px;width:500px;max-width:90vw;max-height:80vh;overflow-y:auto}
.modal-title{font-size:18px;font-weight:600;margin-bottom:20px}
.modal-actions{display:flex;gap:8px;justify-content:flex-end;margin-top:20px}
.stats-grid{display:grid;grid-template-columns:repeat(auto-fit,minmax(200px,1fr));gap:16px;margin-bottom:20px}
.stat-card{background:var(--card);border:1px solid var(--border);border-radius:var(--radius);padding:20px;text-align:center}
.stat-value{font-size:28px;font-weight:700;color:var(--accent)}
.stat-label{font-size:13px;color:var(--muted);margin-top:4px}
.toast{position:fixed;bottom:24px;right:24px;padding:12px 20px;border-radius:var(--radius);font-size:14px;z-index:200;animation:slideIn .3s}
.toast-success{background:var(--green);color:#fff}
.toast-error{background:var(--accent);color:#fff}
@keyframes slideIn{from{transform:translateX(100%);opacity:0}to{transform:translateX(0);opacity:1}}
.empty{text-align:center;padding:40px;color:var(--muted)}
.mono{font-family:'Courier New',monospace;font-size:13px}
.test-result{margin-top:12px;padding:12px;border-radius:var(--radius);font-size:13px}
.test-success{background:rgba(78,204,163,.1);border:1px solid rgba(78,204,163,.3)}
.test-error{background:rgba(233,69,96,.1);border:1px solid rgba(233,69,96,.3)}
.loading{display:inline-block;width:14px;height:14px;border:2px solid var(--border);border-top-color:var(--accent);border-radius:50%;animation:spin .6s linear infinite}
@keyframes spin{to{transform:rotate(360deg)}}
.config-box{background:var(--bg);border:1px solid var(--border);border-radius:var(--radius);padding:16px;margin-top:16px}
.config-box h4{font-size:13px;color:var(--muted);margin-bottom:8px}
.config-line{font-family:'Courier New',monospace;font-size:13px;padding:4px 0;color:var(--green)}
</style>
</head>
<body>
<div class="header">
<h1>AI Models <span>Gateway</span></h1>
<div class="info" id="headerInfo">加载中...</div>
</div>
<div class="container">
<div class="tabs">
<button class="tab active" onclick="switchTab('providers')">中转站管理</button>
<button class="tab" onclick="switchTab('keys')">API Keys</button>
<button class="tab" onclick="switchTab('stats')">用量统计</button>
<button class="tab" onclick="switchTab('settings')">设置</button>
<button class="tab" onclick="switchTab('config')">接入配置</button>
</div>

<!-- 中转站管理 -->
<div class="panel active" id="panel-providers">
<div class="card">
<div class="card-title">中转站列表 <button class="btn btn-primary" onclick="showProviderModal()">+ 添加中转站</button></div>
<div id="providerList"></div>
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
<div class="config-line">Base URL: <span id="cfgOpenAIUrl">http://127.0.0.1:3458/v1</span></div>
<div class="config-line">API Key: <span id="cfgOpenAIKey">在 API Keys 页面生成</span></div>
<div class="config-line">Model: <span id="cfgModel">gpt-4o-mini</span></div>
</div>
<div class="config-box">
<h4>Anthropic 格式 (Claude Code / Cline)</h4>
<div class="config-line">Base URL: <span id="cfgAnthropicUrl">http://127.0.0.1:3458</span></div>
<div class="config-line">API Key: <span id="cfgAnthropicKey">在 API Keys 页面生成</span></div>
<div class="config-line">Model: <span id="cfgModel2">gpt-4o-mini</span></div>
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
<select class="select" id="provFormat" onchange="updateModelsPlaceholder()">
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
<input class="input" id="provBaseUrl" placeholder="https://api.openai.com/v1">
</div>
<div class="form-group">
<label>API Key</label>
<input class="input" id="provApiKey" placeholder="sk-..." type="password">
</div>
<div class="form-group">
<label>支持的模型 (逗号分隔)</label>
<textarea class="input" id="provModels" rows="3" placeholder="gpt-4o-mini, gpt-4o, claude-3-5-sonnet"></textarea>
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

<div id="toast"></div>

<script>
const API='/admin/api';
let activeProviderId='';

function switchTab(name){
document.querySelectorAll('.tab').forEach(t=>t.classList.remove('active'));
document.querySelectorAll('.panel').forEach(p=>p.classList.remove('active'));
event.target.classList.add('active');
document.getElementById('panel-'+name).classList.add('active');
if(name==='stats')loadStats();
if(name==='config')loadConfig();
if(name==='settings')loadSettings();
}

// --- Providers ---
async function loadProviders(){
const res=await fetch(API+'/providers');
const data=await res.json();
const el=document.getElementById('providerList');
if(!data||data.length===0){
el.innerHTML='<div class="empty">暂无中转站，点击右上角添加</div>';
document.getElementById('headerInfo').textContent='未配置中转站';
return;
}
// 获取设置中的活跃站点
const sRes=await fetch(API+'/settings');
const settings=await sRes.json();
activeProviderId=settings.activeProviderId||'';

let html='<table><tr><th>名称</th><th>格式</th><th>Base URL</th><th>模型数</th><th>请求数</th><th>Token总量</th><th>状态</th><th>操作</th></tr>';
for(const p of data){
const isActive=p.id===activeProviderId;
html+='<tr>';
html+='<td>'+esc(p.name)+(isActive?' <span class="badge badge-current">当前</span>':'')+'</td>';
html+='<td><span class="badge badge-'+p.format+'">'+p.format+'</span></td>';
html+='<td class="mono">'+esc(p.baseUrl)+'</td>';
html+='<td>'+(p.models?p.models.length:0)+'</td>';
html+='<td>'+p.usageCount+'</td>';
html+='<td>'+(p.totalTokens||0).toLocaleString()+'</td>';
html+='<td><span class="badge badge-'+p.status+'">'+p.status+'</span></td>';
html+='<td>';
if(!isActive&&p.status==='active')html+='<button class="btn btn-sm btn-success" onclick="setActive(\''+p.id+'\')">启用</button> ';
html+='<button class="btn btn-sm btn-outline" onclick="editProvider(\''+p.id+'\')">编辑</button> ';
html+='<button class="btn btn-sm btn-danger" onclick="deleteProvider(\''+p.id+'\')">删除</button>';
html+='</td>';
html+='</tr>';
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

function showProviderModal(p){
document.getElementById('providerModalTitle').textContent='添加中转站';
document.getElementById('provId').value='';
document.getElementById('provName').value='';
document.getElementById('provFormat').value='openai';
document.getElementById('provStatus').value='active';
document.getElementById('provBaseUrl').value='';
document.getElementById('provApiKey').value='';
document.getElementById('provModels').value='';
document.getElementById('testResult').innerHTML='';
document.getElementById('providerModal').classList.add('show');
}

async function editProvider(id){
const res=await fetch(API+'/providers');
const data=await res.json();
const p=data.find(x=>x.id===id);
if(!p)return;
document.getElementById('providerModalTitle').textContent='编辑中转站';
document.getElementById('provId').value=p.id;
document.getElementById('provName').value=p.name;
document.getElementById('provFormat').value=p.format;
document.getElementById('provStatus').value=p.status;
document.getElementById('provBaseUrl').value=p.baseUrl;
document.getElementById('provApiKey').value=p.apiKey;
document.getElementById('provModels').value=(p.models||[]).join(', ');
document.getElementById('testResult').innerHTML='';
document.getElementById('providerModal').classList.add('show');
}

async function saveProvider(){
const id=document.getElementById('provId').value;
const models=document.getElementById('provModels').value.split(',').map(s=>s.trim()).filter(s=>s);
const body={
name:document.getElementById('provName').value,
format:document.getElementById('provFormat').value,
status:document.getElementById('provStatus').value,
baseUrl:document.getElementById('provBaseUrl').value,
apiKey:document.getElementById('provApiKey').value,
models:models
};
if(!body.name||!body.baseUrl){
toast('请填写名称和 Base URL','error');return;
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

function updateModelsPlaceholder(){
const fmt=document.getElementById('provFormat').value;
const ta=document.getElementById('provModels');
if(fmt==='anthropic'){
ta.placeholder='claude-3-5-sonnet-20241022, claude-3-opus';
}else{
ta.placeholder='gpt-4o-mini, gpt-4o, deepseek-chat';
}
}

async function testProvider(){
const btn=document.getElementById('testBtn');
const result=document.getElementById('testResult');
btn.innerHTML='<span class="loading"></span> 测试中...';
btn.disabled=true;
result.innerHTML='';
const body={
baseUrl:document.getElementById('provBaseUrl').value,
apiKey:document.getElementById('provApiKey').value,
format:document.getElementById('provFormat').value,
model:document.getElementById('provModels').value.split(',')[0].trim()||'gpt-4o-mini'
};
if(!body.baseUrl||!body.apiKey){
result.innerHTML='<div class="test-error">请填写 Base URL 和 API Key</div>';
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

// 按站点
let bp='<table><tr><th>站点</th><th>请求数</th><th>输入Token</th><th>输出Token</th><th>总量</th></tr>';
const providers=stats.byProvider||{};
for(const[name,v]of Object.entries(providers)){
bp+='<tr><td>'+esc(name)+'</td><td>'+v.count+'</td><td>'+v.input+'</td><td>'+v.output+'</td><td>'+(v.total||0).toLocaleString()+'</td></tr>';
}
if(Object.keys(providers).length===0)bp+='<tr><td colspan="5" class="empty">暂无数据</td></tr>';
bp+='</table>';
document.getElementById('statsByProvider').innerHTML=bp;

// 按模型
let bm='<table><tr><th>模型</th><th>请求数</th><th>输入Token</th><th>输出Token</th><th>总量</th></tr>';
const models=stats.byModel||{};
for(const[name,v]of Object.entries(models)){
bm+='<tr><td class="mono">'+esc(name)+'</td><td>'+v.count+'</td><td>'+v.input+'</td><td>'+v.output+'</td><td>'+(v.total||0).toLocaleString()+'</td></tr>';
}
if(Object.keys(models).length===0)bm+='<tr><td colspan="5" class="empty">暂无数据</td></tr>';
bm+='</table>';
document.getElementById('statsByModel').innerHTML=bm;

// 日志
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
}

// --- Utils ---
function closeModal(id){document.getElementById(id).classList.remove('show');}
function esc(s){return String(s||'').replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;').replace(/"/g,'&quot;');}
function fmtDate(s){if(!s)return'-';const d=new Date(s);return d.toLocaleString('zh-CN',{month:'2-digit',day:'2-digit',hour:'2-digit',minute:'2-digit'});}
function toast(msg,type){
const el=document.getElementById('toast');
el.className='toast toast-'+type;
el.textContent=msg;
setTimeout(()=>{el.className='';},3000);
}

// 点击遮罩关闭
document.querySelectorAll('.modal-overlay').forEach(o=>{o.addEventListener('click',e=>{if(e.target===o)o.classList.remove('show');});});

// 初始化
loadProviders();
loadKeys();
</script>
</body>
</html>`
