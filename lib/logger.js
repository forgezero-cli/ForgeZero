class Logger {
  constructor(verbose = false) {
    this.verbose = verbose;
  }

  info(msg) {
    console.log(msg);
  }

  debug(msg) {
    if (this.verbose) console.log(`[DEBUG] ${msg}`);
  }

  error(msg) {
    console.error(`[ERROR] ${msg}`);
  }

  warn(msg) {
    console.warn(`[WARN] ${msg}`);
  }
}

module.exports = { Logger };
