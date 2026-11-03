// Embedded Node adapter. Builtin mode never loads project JavaScript configs.
'use strict';
const fs = require('node:fs');
const path = require('node:path');
const { createRequire } = require('node:module');
const input = JSON.parse(fs.readFileSync(0, 'utf8'));
const { root, files, config, check: name } = input;
const req = createRequire(path.join(config.tool_dir, 'package.json'));
const result = { tool: name, version: '', node_version: process.version, status: 'passed',
  analyzed: 0, total: files.length, diagnostics: [], metrics: {}, files: [], configuration_files: [] };
const selected = new Set(files);
const absolute = files.map(file => path.join(root, file));
console.log = (...args) => console.error(...args);
const canonical = file => path.resolve(file);
function relative(file) {
  const rel = path.relative(root, canonical(file)).split(path.sep).join('/');
  return rel === '..' || rel.startsWith('../') ? null : rel;
}
function remember(file) { if (file && !result.configuration_files.includes(canonical(file))) result.configuration_files.push(canonical(file)); }
function finding(rule, severity, file, line, column, message, toolSeverity = '') {
  const rel = relative(file);
  if (rel === null || !selected.has(rel)) return;
  result.diagnostics.push({tool:name, rule, severity, tool_severity:toolSeverity,
    file:rel, line:line || 1, column:column || 1, message});
}
function mark(file) {
  const rel = relative(file);
  if (rel !== null && selected.has(rel) && !result.files.includes(rel)) result.files.push(rel);
}
const extensions = ['js','jsx','mjs','cjs','ts','tsx','mts','cts'];
const allPatterns = [`**/*.{${extensions.join(',')}}`];
function builtinConfig(typed, projects) {
  const js = req('@eslint/js');
  const tse = req('typescript-eslint');
  const react = req('eslint-plugin-react');
  const globals = req('globals');
  const env = config.environment === 'node' ? globals.node : config.environment === 'browser' ? globals.browser : {...globals.node,...globals.browser};
  const rules = typed ? {
    '@typescript-eslint/no-floating-promises':'error',
    '@typescript-eslint/no-misused-promises':'error',
    '@typescript-eslint/await-thenable':'error'
  } : {...js.configs.recommended.rules,
    'no-unused-vars':['warn',{argsIgnorePattern:'^_'}],
    complexity:['warn',config.complexity_limit],
    'react/jsx-uses-vars':'error', 'react/jsx-uses-react':'error'};
  const parserOptions = {ecmaFeatures:{jsx:true}};
  if (typed) { parserOptions.project = projects; parserOptions.tsconfigRootDir = root; }
  const settings = [{files:allPatterns,
    languageOptions:{parser:tse.parser, parserOptions, ecmaVersion:'latest',sourceType:'module',globals:env},
    plugins:{'@typescript-eslint':tse.plugin,react}, rules}];
  settings.push({files:['**/*.cjs','**/*.cts'],languageOptions:{sourceType:'commonjs'}});
  if (!typed) settings.push({files:['**/*.ts','**/*.tsx','**/*.mts','**/*.cts'],rules:{
    'no-undef':'off','no-unused-vars':'off',
    '@typescript-eslint/no-unused-vars':['warn',{argsIgnorePattern:'^_'}]}});
  return settings;
}
async function lint(typed=false) {
  const { ESLint } = req('eslint');
  result.version = `${ESLint.version}; typescript-eslint ${req('typescript-eslint/package.json').version}`;
  let projects;
  if (typed) projects = collectProjects().projects;
  const options = {cwd:root,fix:false,cache:false,warnIgnored:true,errorOnUnmatchedPattern:true};
  if (config.config_mode === 'builtin' || typed) {
    options.overrideConfigFile = true;
    options.overrideConfig = builtinConfig(typed, projects);
  }
  const engine = new ESLint(options);
  const items = await engine.lintFiles(absolute);
  const serious = new Set(['no-undef','no-unreachable','no-dupe-args','no-dupe-keys','no-dupe-class-members',
    'no-const-assign','no-class-assign','no-func-assign','constructor-super','valid-typeof',
    '@typescript-eslint/no-floating-promises','@typescript-eslint/no-misused-promises']);
  let suppressed = 0;
  for (const item of items) {
    if (config.config_mode === 'project' && !typed) remember(await engine.findConfigFile(item.filePath));
    suppressed += (item.suppressedMessages || []).length;
    if (item.fatalErrorCount || item.messages.some(d => d.ruleId === null)) {
      result.status='error';result.error='Parser/configuration failure or selected file was ignored';
    } else mark(item.filePath);
    for (const d of item.messages) finding(d.ruleId || 'parser-or-configuration', (serious.has(d.ruleId) || (config.config_mode==='project' && !typed && d.severity===2)) ? 'error':'warning',
      item.filePath,d.line,d.column,d.message,d.severity===2?'error':'warning');
  }
  result.metrics.suppressed_diagnostics=suppressed;
  result.analyzed=result.files.length;
}
async function formatting() {
  const prettier = req('prettier'); result.version=prettier.version;
  for (const file of absolute) {
    let options={};
    if (config.config_mode==='project') {
      remember(await prettier.resolveConfigFile(file));
      const info=await prettier.getFileInfo(file,{ignorePath:path.join(root,'.prettierignore')});
      remember(fs.existsSync(path.join(root,'.prettierignore')) ? path.join(root,'.prettierignore') : null);
      if (info.ignored) throw new Error(`Selected file ignored by Prettier: ${relative(file)}`);
      options=await prettier.resolveConfig(file,{editorconfig:true}) || {};
      const editorconfig=path.join(root,'.editorconfig');if(fs.existsSync(editorconfig)) remember(editorconfig);
    }
    const text=fs.readFileSync(file,'utf8');
    if (!await prettier.check(text,{...options,filepath:file})) finding('format','warning',file,1,1,'File differs from the configured Prettier format');
    mark(file);
  }
  result.analyzed=result.files.length;
}
function collectProjects() {
  const ts=req('typescript');
  if (config.config_mode==='project') {
    const projectReq=createRequire(path.join(root,'package.json'));
    const projectTS=projectReq('typescript');
    if (projectTS.version!==ts.version) throw new Error(`Project TypeScript ${projectTS.version} differs from toolkit ${ts.version}; install matching analyzer tools`);
  }
  const projects=[], configurations=[], visited=new Set();
  function walk(candidate) {
    let file=canonical(candidate);
    if (fs.existsSync(file) && fs.statSync(file).isDirectory()) file=path.join(file,'tsconfig.json');
    if (visited.has(file)) return;
    visited.add(file); remember(file);
    const read=ts.readConfigFile(file,ts.sys.readFile);
    if(read.error) throw new Error(ts.flattenDiagnosticMessageText(read.error.messageText,'\n'));
    const host={...ts.sys,readFile:p=>{remember(p);return ts.sys.readFile(p)}};
    const parsed=ts.parseJsonConfigFileContent(read.config,host,path.dirname(file),undefined,file);
    if(parsed.errors.length) throw new Error(parsed.errors.map(d=>ts.flattenDiagnosticMessageText(d.messageText,'\n')).join('\n'));
    configurations.push(parsed);
    if(parsed.fileNames.length) projects.push(file);
    for(const reference of parsed.projectReferences || []) walk(reference.path);
  }
  for(const entry of config.tsconfigs) walk(path.resolve(root,entry));
  if(!projects.length) throw new Error('No source projects in tsconfig/reference graph');
  return {ts,projects,configurations};
}
function types() {
  const {ts,projects,configurations}=collectProjects();result.version=ts.version;
  let strict=0;
  const programFiles=new Set();
  const environmentalCodes=new Set([2307,2688,5058,6053,6059,6305]);
  for(const parsed of configurations) {
    if(!parsed.fileNames.length) continue;
    if(parsed.options.strict) strict++;
    // Do not call emit/build: analysis never writes JS, declarations or tsbuildinfo.
    const program=ts.createProgram({rootNames:parsed.fileNames,options:{...parsed.options,noEmit:true},projectReferences:parsed.projectReferences});
    for(const source of program.getSourceFiles()) {
      const rel=relative(source.fileName);
      if(rel===null || !selected.has(rel)) continue;
      programFiles.add(rel);
      const js=/\.(?:[cm]?js|jsx)$/.test(source.fileName);
      const directive=source.checkJsDirective;
      const enabled=directive ? directive.enabled : (!js || !!parsed.options.checkJs);
      if(!parsed.options.noCheck && enabled) mark(source.fileName);
    }
    for(const d of ts.getPreEmitDiagnostics(program)) {
      const message=ts.flattenDiagnosticMessageText(d.messageText,'\n');
      if(!d.file || environmentalCodes.has(d.code)) {result.status='error';result.error=`TypeScript configuration/dependency error TS${d.code}: ${message}`;}
      if(d.file) {
        const rel=relative(d.file.fileName);
        if(rel===null || !selected.has(rel)) {
          if(d.category===ts.DiagnosticCategory.Error) {result.status='error';result.error=`TypeScript diagnostic outside selected source: ${d.file.fileName}: TS${d.code}`;}
          continue;
        }
        const pos=ts.getLineAndCharacterOfPosition(d.file,d.start || 0);
        finding(`TS${d.code}`,d.category===ts.DiagnosticCategory.Error?'error':'warning',d.file.fileName,pos.line+1,pos.character+1,message);
      }
    }
  }
  result.analyzed=result.files.length;
  const missing=files.filter(f=>!result.files.includes(f));
  if(missing.length) {result.status='error';result.error=`Selected files not type-checked (missing from programs, checkJs disabled or ts-nocheck): ${missing.join(', ')}`;}
  result.metrics={projects:projects.length,strict_projects:strict,selected_files:files.length,covered_files:result.files.length,program_files:programFiles.size};
}
(async()=>{
  try {
    if(name==='eslint') await lint();
    else if(name==='prettier') await formatting();
    else if(name==='typescript') types();
    else if(name==='typed-eslint') await lint(true);
    else throw new Error(`Unknown check ${name}`);
    if(result.status!=='error' && result.analyzed!==result.total) {result.status='error';result.error='Selected files were not completely analyzed';}
    if(result.status!=='error' && result.diagnostics.length) result.status='findings';
  } catch(error) {result.status='error';result.error=error.stack || String(error);}
  // TypeScript project services may keep watcher handles alive. This worker
  // serves exactly one request; exit only after the JSON has been flushed.
  process.stdout.write(JSON.stringify(result), () => process.exit(0));
})();
