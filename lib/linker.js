const { exec } = require('child_process');
const util = require('util');
const execPromise = util.promisify(exec);

class LinuxLinker {
  async link(objFiles, output, linkFlags, linkLibs) {
    let cmd = `ld ${linkFlags}`;
    for (const lib of linkLibs) cmd += ` -l${lib}`;
    cmd += ` ${objFiles.map(f => `"${f}"`).join(' ')} -o "${output}"`;
    if (!linkFlags.includes('-dynamic-linker')) {
      cmd += ' -dynamic-linker /lib64/ld-linux-x86-64.so.2';
    }
    await execPromise(cmd);
  }
}

class MacLinker {
  async link(objFiles, output, linkFlags, linkLibs) {
    let cmd = `ld ${linkFlags}`;
    for (const lib of linkLibs) cmd += ` -l${lib}`;
    cmd += ` ${objFiles.map(f => `"${f}"`).join(' ')} -o "${output}"`;
    await execPromise(cmd);
  }
}

class WindowsLinker {
  async link(objFiles, output, linkFlags, linkLibs) {
    let cmd = `gcc ${linkFlags}`;
    for (const lib of linkLibs) cmd += ` -l${lib}`;
    cmd += ` ${objFiles.map(f => `"${f}"`).join(' ')} -o "${output}"`;
    await execPromise(cmd);
  }
}

function createLinker(platform) {
  switch (platform) {
    case 'linux': return new LinuxLinker();
    case 'darwin': return new MacLinker();
    case 'win32': return new WindowsLinker();
    default: throw new Error(`Unsupported platform: ${platform}`);
  }
}

module.exports = { createLinker };
