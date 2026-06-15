const fs = require('fs');
const glob = require('glob');

// Fix apiService imports
const files = glob.sync('src/**/*.{ts,tsx}', { cwd: 'd:/Demo/AI-DESK/frontend', absolute: true });
files.forEach(f => {
  let c = fs.readFileSync(f, 'utf8');
  if (c.includes("import apiService from")) {
    c = c.replace(/import apiService from ['"]@\/services\/api['"];?/g, "import { apiService } from '@/services/api';");
    fs.writeFileSync(f, c);
  }
});

// Fix EngineersPage.tsx status types
let eng = fs.readFileSync('d:/Demo/AI-DESK/frontend/src/pages/EngineersPage.tsx', 'utf8');
eng = eng.replace(/const\s+statusColors\s*=\s*\{/g, "const statusColors: Record<string, string> = {");
eng = eng.replace(/<span\s+className=\{`\$\{statusColors\[engineer\.status \|\| 'active'\]\}\s+px-2/g, "<span className={`px-2 inline-flex text-xs leading-5 font-semibold rounded-full ${statusColors[engineer.status || 'active'] || statusColors.active}`}");
eng = eng.replace(/statusColors\[engineer\.status \|\| 'active'\]/g, "(statusColors[engineer.status || 'active'] || statusColors.active)");
fs.writeFileSync('d:/Demo/AI-DESK/frontend/src/pages/EngineersPage.tsx', eng);

console.log('Fixed apiService imports and EngineersPage types');
