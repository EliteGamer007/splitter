import { neon } from '@neondatabase/serverless';
import { readFileSync } from 'fs';
const sql = neon('postgres://neondb_owner:npg_mV2byJtuqlc1@ep-falling-mode-a1k832j8-pooler.ap-southeast-1.aws.neon.tech/neondb_2?sslmode=require');
const migSQL = readFileSync(new URL('../migrations/021_ai_moderation.sql', import.meta.url), 'utf8');
const statements = migSQL.split(';').map(s => { const lines = s.split('\n').filter(l => !l.trim().startsWith('--')).join('\n').trim(); return lines; }).filter(s => s.length > 0);
for (const stmt of statements) {
  const preview = stmt.replace(/\s+/g, ' ').substring(0, 60);
  process.stdout.write(preview + '... ');
  try { await sql.query(stmt); console.log('OK'); }
  catch (e) {
    if (e.message.includes('already exists') || e.message.includes('does not exist')) console.log('SKIP');
    else { console.error('FAIL:', e.message); process.exit(1); }
  }
}
console.log('Instance 2 migration done.');
