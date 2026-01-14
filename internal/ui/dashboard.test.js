/* global process */
// Unit tests for dashboard filter functions
// Run with: node internal/ui/dashboard.test.js

var passed = 0
var failed = 0

function assert(condition, message) {
    if (condition) {
        passed++
        console.log('✓', message)
    } else {
        failed++
        console.log('✗', message)
    }
}

function assertEqual(actual, expected, message) {
    if (actual === expected) {
        passed++
        console.log('✓', message)
    } else {
        failed++
        console.log('✗', message, '- expected:', expected, 'got:', actual)
    }
}

// Copy of functions from dashboard.js for testing
function normalizeForSearch(text) {
    return text.toLowerCase().replace(/[-_ ]/g, '')
}

function matchesFilter(text, normalizedQuery) {
    if (!normalizedQuery) return true
    return normalizeForSearch(text).indexOf(normalizedQuery) !== -1
}

// Tests for normalizeForSearch
console.log('\n=== normalizeForSearch ===')
assertEqual(normalizeForSearch('hello'), 'hello', 'lowercase passthrough')
assertEqual(normalizeForSearch('Hello World'), 'helloworld', 'removes spaces, lowercases')
assertEqual(normalizeForSearch('hello-world'), 'helloworld', 'removes dashes')
assertEqual(normalizeForSearch('hello_world'), 'helloworld', 'removes underscores')
assertEqual(normalizeForSearch('hello-world_foo bar'), 'helloworldfoobar', 'removes all separators')
assertEqual(normalizeForSearch('Android-Assistant'), 'androidassistant', 'real example')

// Tests for matchesFilter
console.log('\n=== matchesFilter ===')
assert(matchesFilter('android-assistant', 'androidassist'), 'android assist matches android-assistant')
assert(matchesFilter('android-assistant', 'android'), 'partial match at start')
assert(matchesFilter('android-assistant', 'assistant'), 'partial match at end')
assert(matchesFilter('android-assistant', 'oidass'), 'partial match in middle')
assert(matchesFilter('My Game App', 'game'), 'match in middle with spaces')
assert(matchesFilter('foo-bar_baz qux', 'barbaz'), 'cross-separator match')
assert(!matchesFilter('hello', 'world'), 'non-match returns false')
assert(matchesFilter('anything', ''), 'empty query matches everything')

// Edge cases
console.log('\n=== Edge Cases ===')
assert(matchesFilter('a-b-c', 'abc'), 'multiple dashes')
assert(matchesFilter('a_b_c', 'abc'), 'multiple underscores')
assert(matchesFilter('a b c', 'abc'), 'multiple spaces')
assert(matchesFilter('a-b_c d', 'abcd'), 'mixed separators')
assertEqual(normalizeForSearch(''), '', 'empty string')
assertEqual(normalizeForSearch('   '), '', 'only spaces')
assertEqual(normalizeForSearch('---'), '', 'only dashes')

// Real-world examples
console.log('\n=== Real-world Examples ===')
assert(matchesFilter('android-assistant', normalizeForSearch('android assist')), 'query: "android assist"')
assert(matchesFilter('android-assistant', normalizeForSearch('android-assist')), 'query: "android-assist"')
assert(matchesFilter('android-assistant', normalizeForSearch('android_assist')), 'query: "android_assist"')
assert(matchesFilter('my-cool-app', normalizeForSearch('my cool')), 'query: "my cool"')
assert(matchesFilter('foo_bar_service', normalizeForSearch('bar service')), 'query: "bar service"')

// Summary
console.log('\n=== Summary ===')
console.log('Passed:', passed)
console.log('Failed:', failed)
process.exit(failed > 0 ? 1 : 0)
