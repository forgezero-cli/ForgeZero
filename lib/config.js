const os = require('os');

const DEFAULTS = {
  assembler: 'nasm',     // nasm or gas
  format: 'elf64',       // for nasm only
  output: null,          // auto-generated
  linkFlags: '',
  linkLibs: [],
  includeDirs: [],
  clean: false,
  verbose: false,
  platform: os.platform()
};

module.exports = { DEFAULTS };

