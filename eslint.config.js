import eslintConfigPrettier from 'eslint-config-prettier'

export default [
    {
        files: ['internal/**/*.js'],
        languageOptions: {
            ecmaVersion: 2020,
            sourceType: 'script',
            globals: {
                // Browser globals
                window: 'readonly',
                document: 'readonly',
                fetch: 'readonly',
                setTimeout: 'readonly',
                setInterval: 'readonly',
                clearTimeout: 'readonly',
                console: 'readonly',
                navigator: 'readonly',
                EventSource: 'readonly',
                Set: 'readonly',
                Promise: 'readonly',
                Date: 'readonly',
                HTMLElement: 'readonly',
                location: 'readonly',
                // UMD module globals (for bundled libraries like morphdom)
                module: 'readonly',
                define: 'readonly',
                exports: 'readonly',
                self: 'readonly',
                // morphdom (bundled in dashboard.js)
                morphdom: 'readonly',
                // Template-injected globals
                TLD: 'readonly',
                PORT: 'readonly',
                INITIAL_DATA: 'readonly',
                ICONS: 'readonly',
            },
        },
        rules: {
            // Functions called from HTML onclick handlers appear unused; tt/iconBtn are utilities
            'no-unused-vars': [
                'warn',
                {
                    args: 'none',
                    varsIgnorePattern: '^(toggle|copy|fix|clear|open|restart|handle|do)[A-Z]|^(tt|iconBtn)$|^_',
                    caughtErrors: 'none',
                },
            ],
            'no-undef': 'error',
            'no-redeclare': 'warn',
            eqeqeq: ['warn', 'smart'],
            semi: ['error', 'never'],
        },
    },
    eslintConfigPrettier,
]
