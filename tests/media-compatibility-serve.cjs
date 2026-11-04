// Fast disposable production serve diagnosis; not installed LocalService proof.
const fs=require('fs'),os=require('os'),path=require('path'),assert=require('assert/strict');
const {spawn,spawnSync}=require('child_process');
(async()=>{
 const root=fs.mkdtempSync(path.join(os.tmpdir(),'m6-serve-check-')),state=path.join(root,'state');let app;
 try{
  let occupied=false;try{await fetch('http://127.0.0.1:2283/health');occupied=true}catch{}assert(!occupied,'Disposable port must be unused');
  app=spawn(process.env.IMMICH_COMPAT_BINARY,['-mode','serve','-data',state,'-logs',path.join(root,'logs')],{stdio:'inherit'});
  for(let i=0;i<100;i++){assert(app.exitCode===null,'App exited');try{if((await fetch('http://127.0.0.1:2283/health')).ok)break}catch{}await new Promise(r=>setTimeout(r,100))}
  const result=spawnSync(process.execPath,['tests/media-compatibility-installed.cjs'],{env:{...process.env,IMMICH_TEST_STATE:state,IMMICH_COMPAT_INSTALLED:'1'},stdio:'inherit',timeout:150000});
  assert.equal(result.status,0,'Standalone production serve matrix failed');
 }finally{if(app&&app.exitCode===null){app.kill();await new Promise((resolve,reject)=>{const timer=setTimeout(()=>reject(Error('App did not stop')),10000);app.once('exit',()=>{clearTimeout(timer);resolve()})})}fs.rmSync(root,{recursive:true,force:true})}
})().catch(e=>{console.error(e.message);process.exit(1)});
