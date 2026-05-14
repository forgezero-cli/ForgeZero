//  Copyright (C) 2026  Alex Voste
//  All rights reserved. Unauthorized access is a violation of protocol.


const fs = require('fs');
const path = require('path');
const { execSync } = require('child_process');

class SuperAsmBuilder {
  constructor() {
    this.platform = process.platform;
    this.args = process.argv.slice(2);
    this.options = {
      assembler: 'nasm',
      format: 'elf64',
      output: null,
      linkFlags: '',
      linkLibs: [],
      includeDirs: [],
      clean: false,
      verbose: false,
      inputFile: null
    };
  }

  showHelp() {
    console.log('B-Asm - A unique NodeJS assembler builder with automatic linking');
    console.log('Author: Wienton Weslov - software engineer and assembly language enthusiast');
    console.log('License: MIT');
    console.log('');
    console.log('Usage:');
    console.log('  node builder.js [options] [input.file]');
    console.log('');
    console.log('Options:');
    console.log('  -h, --help              Show this help message');
    console.log('  --assembler <asm>       Assembler to use: nasm or gas (default: nasm)');
    console.log('  --format <format>       NASM output format (default: elf64, ignored for gas)');
    console.log('  --output <file>         Output executable name (default: auto from input)');
    console.log('  --link-flags <flags>    Extra flags for linker (default: auto for platform)');
    console.log('  --link-libs <libs>      Libraries to link (e.g., --link-libs m,c)');
    console.log('  --include <dir>         Add include directory (can be used multiple times)');
    console.log('  --clean                 Remove object files after linking');
    console.log('  --verbose               Show detailed build information');
    console.log('');
    console.log('Supported platforms: Linux (ld), Windows (gcc), macOS (ld)');
    console.log('Detected platform:', this.platform);
    console.log('');
    console.log('Examples:');
    console.log('  # Build specific NASM file');
    console.log('  node builder.js hello.asm');
    console.log('');
    console.log('  # Build all .asm files in current directory');
    console.log('  node builder.js');
    console.log('');
    console.log('  # Build with GAS assembler');
    console.log('  node builder.js --assembler gas hello.s');
    console.log('');
    console.log('  # Build with custom format and output');
    console.log('  node builder.js hello.asm --format elf32 --output hello');
    console.log('');
    console.log('  # Build with libraries');
    console.log('  node builder.js main.asm --link-libs m,c');
    console.log('');
    console.log('  # Build with cleanup and verbose output');
    console.log('  node builder.js --verbose --clean');
  }

  parseArgs() {
    for (let i = 0; i < this.args.length; i++) {
      const arg = this.args[i];
      if (arg === '-h' || arg === '--help') {
        this.showHelp();
        process.exit(0);
      } else if (arg === '--assembler' && this.args[i + 1]) {
        this.options.assembler = this.args[i + 1];
        i++;
      } else if (arg === '--format' && this.args[i + 1]) {
        this.options.format = this.args[i + 1];
        i++;
      } else if (arg === '--output' && this.args[i + 1]) {
        this.options.output = this.args[i + 1];
        i++;
      } else if (arg === '--link-flags' && this.args[i + 1]) {
        this.options.linkFlags = this.args[i + 1];
        i++;
      } else if (arg === '--link-libs' && this.args[i + 1]) {
        this.options.linkLibs = this.args[i + 1].split(',');
        i++;
      } else if (arg === '--include' && this.args[i + 1]) {
        this.options.includeDirs.push(this.args[i + 1]);
        i++;
      } else if (arg === '--clean') {
        this.options.clean = true;
      } else if (arg === '--verbose') {
        this.options.verbose = true;
      } else if (!arg.startsWith('-') && !this.options.inputFile) {
        this.options.inputFile = arg;
      }
    }

    if (!this.options.output) {
      if (this.options.inputFile) {
        const ext = this.options.assembler === 'gas' ? '.s' : '.asm';
        this.options.output = path.basename(this.options.inputFile, ext);
      } else {
        this.options.output = 'a.out';
      }
    }
  }

  findFiles(dir) {
    const ext = this.options.assembler === 'gas' ? '.s' : '.asm';
    const files = [];
    const items = fs.readdirSync(dir);
    for (const item of items) {
      const fullPath = path.join(dir, item);
      const stat = fs.statSync(fullPath);
      if (stat.isDirectory()) {
        files.push(...this.findFiles(fullPath));
      } else if (path.extname(item) === ext) {
        files.push(fullPath);
      }
    }
    return files;
  }

  compile(file) {
    const objFile = file.replace(/\.(asm|s)$/, '.o');
    let cmd;
    if (this.options.assembler === 'nasm') {
      cmd = `nasm -f ${this.options.format}`;
      for (const dir of this.options.includeDirs) {
        cmd += ` -I "${dir}"`;
      }
      cmd += ` "${file}" -o "${objFile}"`;
    } else if (this.options.assembler === 'gas') {
      cmd = `as`;
      for (const dir of this.options.includeDirs) {
        cmd += ` -I "${dir}"`;
      }
      cmd += ` "${file}" -o "${objFile}"`;
    }
    if (this.options.verbose) {
      console.log(`Executing: ${cmd}`);
    }
    execSync(cmd, { stdio: 'inherit' });
    return objFile;
  }

  getDefaultLinker() {
    if (this.platform === 'linux') {
      return 'ld';
    } else if (this.platform === 'win32') {
      return 'gcc';
    } else if (this.platform === 'darwin') {
      return 'ld';
    }
    return 'ld';
  }

  link(objFiles) {
    const linker = this.getDefaultLinker();
    let cmd = `${linker} ${this.options.linkFlags}`;
    for (const lib of this.options.linkLibs) {
      cmd += ` -l${lib}`;
    }
    cmd += ` ${objFiles.map(f => `"${f}"`).join(' ')} -o "${this.options.output}"`;
    if (this.platform === 'linux' && !this.options.linkFlags.includes('-dynamic-linker')) {
      cmd += ' -dynamic-linker /lib64/ld-linux-x86-64.so.2';
    }
    if (this.options.verbose) {
      console.log(`Executing: ${cmd}`);
    }
    execSync(cmd, { stdio: 'inherit' });
  }

  build() {
    if (this.options.verbose) {
      console.log('B-Asm starting build...');
      console.log('Platform:', this.platform);
      console.log('Assembler:', this.options.assembler);
    }

    let asmFiles = [];
    if (this.options.inputFile) {
      const ext = this.options.assembler === 'gas' ? '.s' : '.asm';
      if (fs.existsSync(this.options.inputFile) && path.extname(this.options.inputFile) === ext) {
        asmFiles = [this.options.inputFile];
      } else {
        console.error(`Input file not found or not an ${ext} file`);
        process.exit(1);
      }
    } else {
      asmFiles = this.findFiles('.');
      if (asmFiles.length === 0) {
        console.error(`No ${this.options.assembler === 'gas' ? '.s' : '.asm'} files found`);
        process.exit(1);
      }
    }

    if (this.options.verbose) {
      console.log(`Found ${asmFiles.length} file(s): ${asmFiles.join(', ')}`);
    }

    const objFiles = [];
    for (const file of asmFiles) {
      if (this.options.verbose) {
        console.log(`Compiling ${file}...`);
      }
      const objFile = this.compile(file);
      objFiles.push(objFile);
    }

    if (this.options.verbose) {
      console.log(`Linking ${objFiles.length} object file(s) to ${this.options.output}...`);
    }
    this.link(objFiles);

    if (this.options.clean) {
      if (this.options.verbose) {
        console.log('Cleaning up object files...');
      }
      for (const obj of objFiles) {
        fs.unlinkSync(obj);
      }
    }

    console.log(`B-Asm: Built ${this.options.output}`);
  }

  run() {
    this.parseArgs();
    this.build();
  }
}

const builder = new SuperAsmBuilder();
builder.run();
