const fs = require('fs');
const path = require('path');

function findFiles(rootDir, extension) {
  const results = [];
  const items = fs.readdirSync(rootDir);
  for (const item of items) {
    const fullPath = path.join(rootDir, item);
    const stat = fs.statSync(fullPath);
    if (stat.isDirectory()) {
      results.push(...findFiles(fullPath, extension));
    } else if (path.extname(item) === extension) {
      results.push(fullPath);
    }
  }
  return results;
}

module.exports = { findFiles };
