#!/usr/bin/env node

const { spawn } = require('child_process');
const path = require('path');
const fs = require('fs');

// 获取二进制文件路径
const platform = process.platform;
const binaryName = platform === 'win32' ? 'xz-mcp.exe' : 'xz-mcp';
const binaryPath = path.join(__dirname, '..', 'bin', binaryName);

// 检查二进制文件是否存在
if (!fs.existsSync(binaryPath)) {
  console.error('❌ 二进制文件不存在');
  console.error('   请运行: npm install');
  console.error('   或手动下载: https://github.com/Xuzan9396/xz_mcp/releases\n');
  process.exit(1);
}

// 启动二进制文件
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
  console.error('❌ 启动失败:', err.message);
  process.exit(1);
});

// 处理信号
process.on('SIGINT', () => {
  child.kill('SIGINT');
});

process.on('SIGTERM', () => {
  child.kill('SIGTERM');
});
