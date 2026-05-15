const path = require('path');
const { DEFAULTS } = require('./config');

function showHelp(platform) {
  console.log(`
  ${"\x1b[1m"}B-Asm${"\x1b[0m"} - Modular & Intelligent Assembly Builder

  ${"\x1b[1m"}USAGE:${"\x1b[0m"}
    node index.js [options] <input.file>

  ${"\x1b[1m"}OPTIONS:${"\x1b[0m"}
    -h, --help                Show this help message
    -a, --assembler <asm>     nasm or gas (default: ${DEFAULTS.assembler})
    -f, --format <fmt>        Object format for NASM (default: ${DEFAULTS.format})
    -o, --output <file>       Output executable name
    -l, --link-libs <libs>    Comma-separated libraries (e.g. m,c)
    -I, --include <dir>       Add include directory (repeatable)
    --link-flags <flags>      Raw flags to pass to the linker
    --clean                   Auto-remove object files after successful link
    --verbose                 Show exactly what's happening under the hood

  ${"\x1b[1m"}SYSTEM:${"\x1b[0m"}
    Detected platform: ${"\x1b[36m"}${platform}${"\x1b[0m"}
    Default assembler: ${"\x1b[32m"}${DEFAULTS.assembler}${"\x1b[0m"}

  ${"\x1b[1m"}EXAMPLES:${"\x1b[0m"}
    $ node index.js hello.asm                     # Auto-detect NASM & link
    $ node index.js -a gas -o my_app main.s       # Build with GAS as 'my_app'
    $ node index.js code.asm -l m,pthread --clean # Link math & threads, then clean up

  Submit issues at: https://github.com/alexvoste/B-Asm
  Author: AlexVoste :)
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
