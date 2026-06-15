const fs = require('fs');
let c = fs.readFileSync('d:/Demo/AI-DESK/frontend/src/pages/EngineersPage.tsx', 'utf8');
c = c.replace(/status: 'active' as const as const/g, "status: 'active' as 'active' | 'inactive'");
c = c.replace(/status: 'active' as const/g, "status: 'active' as 'active' | 'inactive'");
c = c.replace(/status: engineer\.status \}/g, "status: engineer.status || 'active' }");
fs.writeFileSync('d:/Demo/AI-DESK/frontend/src/pages/EngineersPage.tsx', c);
console.log('Fixed EngineersPage.tsx final TS errors');
