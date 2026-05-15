const fs = require('fs').promises;
const path = require('path');
const { findFiles } = require('./file-scanner');
const { createAssembler } = require('./assembler');
const { createLinker } = require('./linker');

class Builder {
  constructor(options, logger) {
    this.options = options;
    this.logger = logger;
    this.assembler = createAssembler(
      options.assembler,
      options.format,
      options.includeDirs,
      options.debug || false
    );
    this.linker = createLinker(options.platform);
  }

  async build() {
    this.logger.debug(`Starting build on ${this.options.platform}`);
    this.logger.debug(`Assembler: ${this.options.assembler}`);
    const asmFiles = await this.getAsmFiles();
    if (asmFiles.length === 0) throw new Error(`No ${this.getExtension()} files found`);
    this.logger.debug(`Found files: ${asmFiles.join(', ')}`);

    const objFiles = [];
    for (const file of asmFiles) {
      this.logger.debug(`Compiling ${file}...`);
      const obj = await this.assembler.compile(file);
      objFiles.push(obj);
    }

    this.logger.debug(`Linking to ${this.options.output}...`);
    await this.linker.link(
      objFiles,
      this.options.output,
      this.options.linkFlags,
      this.options.linkLibs,
      this.options.debug || false
    );

    if (this.options.clean) {
      this.logger.debug('Cleaning object files...');
      for (const obj of objFiles) await fs.unlink(obj);
    }

    this.logger.info(`✅ Built ${this.options.output}`);
  }

  async getAsmFiles() {
    const ext = this.getExtension();
    if (this.options.inputFile) {
      const exists = await fs.access(this.options.inputFile).then(() => true).catch(() => false);
      if (!exists || path.extname(this.options.inputFile) !== ext)
        throw new Error(`Input file not found or not a ${ext} file`);
      return [this.options.inputFile];
    }
    return findFiles('.', ext);
  }

  getExtension() {
    const extMap = { nasm: '.asm', gas: '.s', fasm: '.asm' };
    return extMap[this.options.assembler] || '.asm';
  }
}

module.exports = { Builder };
