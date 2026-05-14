const path = require('path');
const { DEFAULTS } = require('./config');

function showHelp(platform) {
  console.log(`
B-Asm - Modular ASM builder
Usage: node index.js [options] [input.file]

Options:
  -h, --help              Show help
  --assembler <asm>       nasm or gas (default: ${DEFAULTS.assembler})
  --format <format>       NASM format (default: ${DEFAULTS.format})
  --output <file>         Output executable name
  --link-flags <flags>    Extra linker flags
  --link-libs <libs>      Comma-separated libs (e.g. m,c)
  --include <dir>         Include directory (repeatable)
  --clean                 Remove .o files after linking
  --verbose               Verbose output

Detected platform: ${platform}
Examples:
  node index.js hello.asm
  node index.js --assembler gas --output prog main.s
`);
}

function parseArgs(argv, logger) {
  const options = { ...DEFAULTS };
  let inputFile = null;

  for (let i = 0; i < argv.length; i++) {
    const arg = argv[i];
    switch (arg) {
      case '-h':
      case '--help':
        showHelp(options.platform);
        options.help = true;
        return options;
      case '--assembler':
        options.assembler = argv[++i];
        break;
      case '--format':
        options.format = argv[++i];
        break;
      case '--output':
        options.output = argv[++i];
        break;
      case '--link-flags':
        options.linkFlags = argv[++i];
        break;
      case '--link-libs':
        options.linkLibs = argv[++i].split(',');
        break;
      case '--include':
        options.includeDirs.push(argv[++i]);
        break;
      case '--clean':
        options.clean = true;
        break;
      case '--verbose':
        options.verbose = true;
        break;
      default:
        if (!arg.startsWith('-') && !inputFile) {
          inputFile = arg;
        }
    }
  }

  if (!options.output) {
    if (inputFile) {
      const ext = options.assembler === 'gas' ? '.s' : '.asm';
      options.output = path.basename(inputFile, ext);
    } else {
      options.output = 'a.out';
    }
  }

  options.inputFile = inputFile;
  logger.verbose = options.verbose;
  return options;
}

module.exports = { parseArgs, showHelp };
