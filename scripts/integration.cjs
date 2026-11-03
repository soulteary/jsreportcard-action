'use strict';
const fs=require('node:fs');
const path=require('node:path');
const os=require('node:os');
const assert=require('node:assert/strict');
const {spawnSync}=require('node:child_process');
const cli=path.resolve(process.argv[2]);
const tools=path.resolve(process.argv[3]);
const clean='export function add(left: number, right: number): number {\n  return left + right;\n}\n';
const root=fs.mkdtempSync(path.join(os.tmpdir(),'jsreportcard-test-'));
function write(file,content){fs.mkdirSync(path.dirname(path.join(root,file)),{recursive:true});fs.writeFileSync(path.join(root,file),content);}
function clear(){for(const name of fs.readdirSync(root))fs.rmSync(path.join(root,name),{recursive:true,force:true});}
function config(value={}){write('tsconfig.json',JSON.stringify({compilerOptions:{strict:true,target:'ES2022',module:'CommonJS'},include:['src'],...value}));}
function check(code=0,args=[],env={}){
 console.log(`Checking expected exit ${code}: ${args.join(' ') || 'standard'}`);
 const response=spawnSync(cli,['--directory',root,'--tool-dir',tools,'--json',...args],{encoding:'utf8',timeout:120000,env:{...process.env,...env}});
 assert.equal(response.status,code,response.stdout+'\n'+response.stderr);
 const report=JSON.parse(response.stdout);
 for(const file of ['report.json','report.md','badge.svg'])assert.ok(fs.existsSync(path.join(root,'.jsreportcard',file)));
 return report;
}
try{
 write('src/index.ts',clean);config();
 let r=check(0,['--mode','typed']);assert.equal(r.grade,'A+');assert.equal(r.analysis_complete,true);
 assert.equal(r.checks.find(c=>c.tool==='typescript').metrics.covered_files,1);
 const combinedProfile=r.analysis_profile;
 for(const [label,text] of [['js','JS Report'],['ts','TS Report']]){
  const selected=check(0,['--mode','typed','--badge-label',label,'--baseline','.jsreportcard/report.json']);
  assert.equal(selected.analysis_profile,combinedProfile);
  assert.equal(selected.grade,'A+');
  assert.equal(selected.badge_label,label);
  assert.ok(fs.readFileSync(path.join(root,'.jsreportcard/badge.svg'),'utf8').includes(`<title>${text} (Typed): A+`));
 }

 assert.ok(!fs.existsSync(path.join(root,'src/index.js')));
 clear();write('src/index.ts','export class Box {\n  value(input: string): string;\n  value(input: number): number;\n  value(input: string | number): string | number {\n    return input;\n  }\n}\n');config();
 r=check(0,['--mode','typed']);assert.ok(!r.checks.flatMap(c=>c.diagnostics).some(d=>d.rule==='no-dupe-class-members'));
 clear();write('src/index.js','console.log(missing);\n');
 r=check(1);assert.equal(r.new_errors,1);assert.equal(r.score,59);write('baseline.json',JSON.stringify(r));
 check(0,['--baseline','baseline.json']);
 write('src/index.js','console.log(missing);\nconsole.log(missing);\n');
 r=check(1,['--baseline','baseline.json']);assert.equal(r.new_errors,1);
 clear();write('src/index.ts','export const value: number = "wrong";\n');config();
 r=check(1,['--mode','typed']);assert.ok(r.checks.flatMap(c=>c.diagnostics).some(d=>d.rule==='TS2322'));
 clear();write('src/index.ts','export function run(): void {\n  Promise.resolve(1);\n}\n');config();
 r=check(1,['--mode','typed']);assert.ok(r.checks.flatMap(c=>c.diagnostics).some(d=>d.rule==='@typescript-eslint/no-floating-promises'));
 clear();write('src/index.js','export function work(x) {\n'+Array.from({length:12},(_,i)=>`  if (x === ${i}) {\n    return ${i};\n  }\n`).join('')+'  return -1;\n}\n');
 r=check();assert.ok(r.checks.flatMap(c=>c.diagnostics).some(d=>d.rule==='complexity'));assert.equal(r.score,80);
 clear();write('src/index.ts','import { value } from "missing-reportcard-dependency";\nexport { value };\n');config();
 r=check(2,['--mode','typed','--fail-on','none']);assert.equal(r.score,null);
 clear();write('src/index.ts',clean);config({files:[],include:[]});
 r=check(2,['--mode','typed','--fail-on','none']);assert.equal(r.analysis_complete,false);
 clear();write('src/index.ts',clean);write('orphan.ts','export const orphan = 1;\n');config();
 r=check(2,['--mode','typed','--fail-on','none']);assert.match(r.checks.find(c=>c.tool==='typescript').error,/not type-checked/);
 clear();write('src/index.js','export const value = 1;\n');config({compilerOptions:{allowJs:true,checkJs:false}});
 r=check(2,['--mode','typed','--fail-on','none']);assert.equal(r.analysis_complete,false);
 clear();write('src/index.js','export const value = 1;\n');config({compilerOptions:{allowJs:true,checkJs:true}});
 r=check(0,['--mode','typed']);assert.equal(r.analysis_complete,true);
 clear();write('src/index.ts','// @ts-nocheck\nexport const value: number = \"wrong\";\n');config();
 r=check(2,['--mode','typed','--fail-on','none']);assert.equal(r.analysis_complete,false);
 clear();write('src/index.js','export const value=1;\n');r=check();assert.equal(r.score,90);assert.match(fs.readFileSync(path.join(root,'src/index.js'),'utf8'),/value=1/);
 clear();write('src/index.js','const = broken;\n');r=check(2,['--fail-on','none']);assert.equal(r.score,null);
 clear();write('src/index.mjs','export const value = 1;\n');write('src/legacy.cjs','module.exports = 1;\n');check(0,['--fail-on','findings']);
 clear();write('src/index.tsx','export const App = () => <div>Hello</div>;\n');
 write('src/types.d.ts','declare namespace JSX { interface IntrinsicElements { div: { children?: unknown } } }\n');
 config({compilerOptions:{strict:true,target:'ES2022',jsx:'preserve'}});r=check(0,['--mode','typed']);assert.equal(r.files,1);
 clear();write('src/index.js','export const value = 1;\n');
 write('eslint.config.cjs',`require('node:fs').writeFileSync(${JSON.stringify(path.join(root,'executed'))},'yes');module.exports=[{files:['**/*.js'],rules:{}}];\n`);
 check(0,['--include','src']);assert.ok(!fs.existsSync(path.join(root,'executed')));
 check(0,['--include','src','--config-mode','project']);assert.ok(fs.existsSync(path.join(root,'executed')));
 write('eslint.config.cjs','throw new Error("bad project configuration");\n');r=check(2,['--include','src','--config-mode','project','--fail-on','none']);assert.equal(r.score,null);
 clear();write('src/index.js','export const value = 1;\n');
 write('preload.cjs',`require('node:fs').writeFileSync(${JSON.stringify(path.join(root,'preloaded'))},'bad');\n`);
 check(0,['--include','src'],{NODE_OPTIONS:`--require=${path.join(root,'preload.cjs')}`});assert.ok(!fs.existsSync(path.join(root,'preloaded')));
 clear();write('tsconfig.json',JSON.stringify({files:[],references:[{path:'packages/a'},{path:'packages/b'}]}));
 for(const name of ['a','b']){
  write(`packages/${name}/index.ts`,clean);
  write(`packages/${name}/tsconfig.json`,JSON.stringify({compilerOptions:{strict:true,composite:true},files:['index.ts']}));
 }
 r=check(0,['--mode','typed','--fail-on','findings']);assert.equal(r.checks.find(c=>c.tool==='typescript').metrics.projects,2);
 console.log('Real JS/TS/TSX, ESM/CJS, baseline, isolation and project-reference tests passed.');
}finally{fs.rmSync(root,{recursive:true,force:true});}
