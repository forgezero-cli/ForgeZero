const { exec } = require('child_process');
const util = require('util');
const fs = require('fs');
const path = require('path');
const execPromise = util.promisify(exec);

class NasmAssembler {
  constructor(format, includeDirs) {
    this.format = format;
    this.includeDirs = includeDirs;
  }

  async compile(sourceFile) {
    const objFile = sourceFile.replace(/\.asm$/i, '.o');
    let cmd = `nasm -f ${this.format}`;
    for (const dir of this.includeDirs) {
      cmd += ` -I "${dir}"`;
    }
    cmd += ` "${sourceFile}" -o "${objFile}"`;
    await execPromise(cmd);
    return objFile;
  }
}

class GasAssembler {
  constructor(includeDirs) {
    this.includeDirs = includeDirs;
  }

  async compile(sourceFile) {
    const objFile = sourceFile.replace(/\.s$/i, '.o');
    let cmd = 'as';
    for (const dir of this.includeDirs) {
      cmd += ` -I "${dir}"`;
    }
    cmd += ` "${sourceFile}" -o "${objFile}"`;
    await execPromise(cmd);
    return objFile;
  }
}

function createAssembler(type, format, includeDirs) {
  if (type === 'nasm') {
    return new NasmAssembler(format, includeDirs);
  } else if (type === 'gas') {
    return new GasAssembler(includeDirs);
  }
  throw new Error(`Unsupported assembler: ${type}`);
}

module.exports = { createAssembler };
