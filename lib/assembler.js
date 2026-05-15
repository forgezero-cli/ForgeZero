const { exec } = require('child_process');
const util = require('util');
const fs = require('fs');
const path = require('path');
const execPromise = util.promisify(exec);

class NasmAssembler {
  constructor(format, includeDirs, debug) {
    this.format = format;
    this.includeDirs = includeDirs;
    this.debug = debug;
  }

  async compile(sourceFile) {
    const objFile = sourceFile.replace(/\.asm$/i, '.o');
    let cmd = `nasm -f ${this.format}`;
    if (this.debug) cmd += ' -g';
    for (const dir of this.includeDirs) cmd += ` -I "${dir}"`;
    cmd += ` "${sourceFile}" -o "${objFile}"`;
    await execPromise(cmd);
    return objFile;
  }
}

class GasAssembler {
  constructor(includeDirs, debug) {
    this.includeDirs = includeDirs;
    this.debug = debug;
  }

  async compile(sourceFile) {
    const objFile = sourceFile.replace(/\.s$/i, '.o');
    let cmd = 'as';
    if (this.debug) cmd += ' -g';
    for (const dir of this.includeDirs) cmd += ` -I "${dir}"`;
    cmd += ` "${sourceFile}" -o "${objFile}"`;
    await execPromise(cmd);
    return objFile;
  }
}

class FasmAssembler {
  constructor(format, includeDirs, debug) {
    this.format = format; // 'elf64' or 'elf32'
    this.includeDirs = includeDirs;
    this.debug = debug;
  }

  async compile(sourceFile) {
    const objFile = sourceFile.replace(/\.fasm$/i, '.o');
    let cmd = `fasm`;
    if (this.debug) cmd += ' -d DEBUG=1';
    for (const dir of this.includeDirs) cmd += ` -I"${dir}"`;
    cmd += ` "${sourceFile}" "${objFile}"`;
    await execPromise(cmd);
    return objFile;
  }
}

function createAssembler(type, format, includeDirs, debug) {
  if (type === 'nasm') return new NasmAssembler(format, includeDirs, debug);
  if (type === 'gas') return new GasAssembler(includeDirs, debug);
  if (type === 'fasm') return new FasmAssembler(format, includeDirs, debug);
  throw new Error(`Unsupported assembler: ${type}`);
}

module.exports = { createAssembler };
