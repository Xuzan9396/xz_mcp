#!/usr/bin/env node

const { spawn } = require('child_process');
const path = require('path');
const fs = require('fs');

// 获取二进制文件路径
const platform = process.platform;
const binaryName = platform === 'win32' ? 'xz_mcp.exe' : 'xz_mcp';
const binaryPath = path.join(__dirname, binaryName);

// 检查二进制文件是否存在
if (!fs.existsSync(binaryPath)) {
  console.error('❌ xz-mcp 二进制文件未找到!');
  console.error('请重新安装: npm install -g @xuzan/xz-mcp');
  process.exit(1);
}

// 启动二进制文件，传递所有参数和环境变量
const child = spawn(binaryPath, process.argv.slice(2), {
  stdio: 'inherit',
  env: process.env
});

// 处理退出
child.on('exit', (code) => {
  process.exit(code || 0);
});

// 处理错误
child.on('error', (err) => {
  console.error('❌ 启动 xz-mcp 失败:', err.message);
  process.exit(1);
});