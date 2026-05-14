#!/usr/bin/env node
const { parseArgs } = require('./lib/args');
const { Logger } = require('./lib/logger');
const { Builder } = require('./lib/builder');

async function main() {
  const logger = new Logger();
  const options = parseArgs(process.argv.slice(2), logger);
  
  if (options.help) return;
  
  const builder = new Builder(options, logger);
  try {
    await builder.build();
  } catch (err) {
    logger.error(`Build failed: ${err.message}`);
    process.exit(1);
  }
}

main();
