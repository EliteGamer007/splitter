import { neon } from '@neondatabase/serverless';
import { readFileSync } from 'fs';

const connStr = process.env.DATABASE_URL;
if (!connStr) throw new Error('DATABASE_URL must be supplied explicitly');

const sql = neon(connStr);

const migrationSQL = readFileSync(new URL('../migrations/021_ai_moderation.sql', import.meta.url), 'utf8');

// Split on semicolons, strip comment lines from each statement, filter blanks
const statements = migrationSQL
  .split(';')
  .map(s => {
    // Remove comment-only lines, keep SQL lines
    const sqlLines = s.split('\n').filter(l => !l.trim().startsWith('--')).join('\n').trim();
    return sqlLines;
  })
  .filter(s => s.length > 0);

console.log(`Running ${statements.length} statements...`);

for (const stmt of statements) {
  const preview = stmt.replace(/\s+/g, ' ').substring(0, 70);
  process.stdout.write(`  ${preview}... `);
  try {
    await sql.query(stmt);
    console.log('OK');
  } catch (e) {
    if (e.message.includes('already exists') || e.message.includes('does not exist')) {
      console.log('SKIP (' + e.message.split('\n')[0] + ')');
    } else {
      console.error('FAIL:', e.message);
      process.exit(1);
    }
  }
}

console.log('\nMigration 021_ai_moderation applied successfully.');
