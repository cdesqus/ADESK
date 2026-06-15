const fs = require('fs');

function replaceInFile(file, searchRegex, replacement) {
  let content = fs.readFileSync(file, 'utf8');
  content = content.replace(searchRegex, replacement);
  fs.writeFileSync(file, content);
}

const filesWithApiImport = [
  'src/hooks/useAuth.ts',
  'src/hooks/useTickets.ts',
  'src/pages/CustomersPage.tsx',
  'src/pages/EngineersPage.tsx',
  'src/pages/TicketDetailPage.tsx'
];

filesWithApiImport.forEach(f => {
  replaceInFile(f, /import api from ['"]@\/services\/api['"];?/g, "import { apiService as api } from '@/services/api';");
});

const filesWithMeta = [
  'src/hooks/useTickets.ts',
  'src/pages/CustomersPage.tsx',
  'src/pages/EngineersPage.tsx'
];

filesWithMeta.forEach(f => {
  replaceInFile(f, /\.meta\./g, ".");
});

replaceInFile('src/types/index.ts', /phone\?: string;/g, "phone?: string;\n  domain?: string;");

replaceInFile('src/pages/EngineersPage.tsx', /engineer\.status === 'active'/g, "(engineer.status || 'active') === 'active'");
replaceInFile('src/pages/EngineersPage.tsx', /status: 'active'/g, "status: 'active' as const");
replaceInFile('src/pages/EngineersPage.tsx', /variant=\{engineer\.status === 'active'/g, "variant={(engineer.status || 'active') === 'active'");

console.log('Fixed more TS errors');
