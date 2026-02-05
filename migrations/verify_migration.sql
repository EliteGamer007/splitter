-- Verification Script - Run this after migration to verify everything is set up correctly
-- This checks that all required tables, columns, indexes, and constraints exist

\echo '==============================================='
\echo 'SPLITTER DATABASE VERIFICATION'
\echo '==============================================='
\echo ''

\echo '1. Checking Tables...'
SELECT 
    CASE 
        WHEN COUNT(*) >= 15 THEN '✓ PASS - Found ' || COUNT(*) || ' tables'
        ELSE '✗ FAIL - Only found ' || COUNT(*) || ' tables (expected 15+)'
    END as table_check
FROM information_schema.tables 
WHERE table_schema = 'public';

\echo ''
\echo '2. Table List:'
SELECT '  - ' || table_name as tables
FROM information_schema.tables 
WHERE table_schema = 'public' 
ORDER BY table_name;

\echo ''
\echo '3. Checking Users Table Columns...'
SELECT 
    '  ' || column_name || ' (' || data_type || 
    CASE WHEN is_nullable = 'NO' THEN ', NOT NULL' ELSE '' END || ')' as user_columns
FROM information_schema.columns
WHERE table_name = 'users'
ORDER BY ordinal_position;

\echo ''
\echo '4. Verifying Critical Columns...'
SELECT 
    CASE 
        WHEN EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='users' AND column_name='email') 
        THEN '  ✓ users.email exists'
        ELSE '  ✗ users.email MISSING'
    END;
SELECT 
    CASE 
        WHEN EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='users' AND column_name='password_hash') 
        THEN '  ✓ users.password_hash exists'
        ELSE '  ✗ users.password_hash MISSING'
    END;
SELECT 
    CASE 
        WHEN EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='users' AND column_name='role') 
        THEN '  ✓ users.role exists'
        ELSE '  ✗ users.role MISSING'
    END;
SELECT 
    CASE 
        WHEN EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='users' AND column_name='moderation_requested') 
        THEN '  ✓ users.moderation_requested exists'
        ELSE '  ✗ users.moderation_requested MISSING'
    END;

\echo ''
\echo '5. Checking Indexes...'
SELECT 
    CASE 
        WHEN COUNT(*) >= 20 THEN '✓ PASS - Found ' || COUNT(*) || ' indexes'
        ELSE '⚠ WARNING - Only found ' || COUNT(*) || ' indexes (expected 20+)'
    END as index_check
FROM pg_indexes 
WHERE schemaname = 'public';

\echo ''
\echo '6. Critical Indexes:'
SELECT 
    '  - ' || indexname || ' on ' || tablename as indexes
FROM pg_indexes 
WHERE schemaname = 'public' 
  AND indexname IN (
      'idx_users_username', 
      'idx_users_email', 
      'idx_posts_author',
      'idx_messages_thread',
      'idx_follows_follower_status'
  )
ORDER BY indexname;

\echo ''
\echo '7. Checking Foreign Keys...'
SELECT 
    '  - ' || conname || ' on ' || conrelid::regclass::text as foreign_keys
FROM pg_constraint
WHERE contype = 'f' AND connamespace = 'public'::regnamespace
LIMIT 10;

\echo ''
\echo '8. Checking Triggers...'
SELECT 
    '  - ' || trigger_name || ' on ' || event_object_table as triggers
FROM information_schema.triggers
WHERE trigger_schema = 'public';

\echo ''
\echo '9. Checking Extensions...'
SELECT 
    '  - ' || extname || ' (version ' || extversion || ')' as extensions
FROM pg_extension
WHERE extname != 'plpgsql';

\echo ''
\echo '10. Database Statistics:'
SELECT 
    'Total Tables: ' || COUNT(DISTINCT table_name)
FROM information_schema.tables 
WHERE table_schema = 'public';

SELECT 
    'Total Columns: ' || COUNT(*)
FROM information_schema.columns 
WHERE table_schema = 'public';

SELECT 
    'Total Indexes: ' || COUNT(*)
FROM pg_indexes 
WHERE schemaname = 'public';

\echo ''
\echo '==============================================='
\echo 'VERIFICATION COMPLETE'
\echo '==============================================='
\echo ''
\echo 'If all checks show ✓ PASS, your database is ready!'
\echo 'If any checks show ✗ FAIL, re-run the migration.'
\echo ''
