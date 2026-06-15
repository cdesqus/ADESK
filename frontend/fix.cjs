const fs = require('fs');
const glob = require('glob');
const files = glob.sync('src/**/*.{ts,tsx}', { cwd: 'd:/Demo/AI-DESK/frontend', absolute: true });
files.forEach(f => {
  let c = fs.readFileSync(f, 'utf8');
  c = c.replace(/import api from ['"]@\/services\/api['"]/g, "import { apiService as api } from '@/services/api'");
  c = c.replace(/import App from '\.\/App\.tsx'/g, "import App from './App'");
  c = c.replace(/\.meta\.page/g, ".page");
  c = c.replace(/\.meta\.totalPages/g, ".totalPages");
  fs.writeFileSync(f, c);
});
console.log('Fixed imports and pagination fields');
