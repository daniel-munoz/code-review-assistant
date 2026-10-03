package parser

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Moved verbatim from ast_test.go TestPatternMatching (Go matcher).
func TestMatchPattern_FormerGoCases(t *testing.T) {
	cases := []struct {
		path, pattern string
		want          bool
	}{
		{"vendor/foo.go", "vendor/**", true},
		{"pkg/vendor/foo.go", "vendor/**", false},
		{"vendor/foo.go", "**/vendor/**", true},
		{"pkg/vendor/foo.go", "**/vendor/**", true},
		{"foo_test.go", "**/*_test.go", true},
		{"pkg/foo_test.go", "**/*_test.go", true},
		{"foo.go", "**/*_test.go", false},
		{"testdata/sample.go", "**/testdata/**", true},
		{"pkg/testdata/sample.go", "**/testdata/**", true},
		{"foo.pb.go", "**/*.pb.go", true},
		{"pkg/foo.pb.go", "**/*.pb.go", true},
		{"foo.go", "**/*.pb.go", false},
	}
	for _, c := range cases {
		assert.Equal(t, c.want, MatchPattern(c.path, c.pattern), "MatchPattern(%q, %q)", c.path, c.pattern)
	}
}

// Moved verbatim from javascript/parser_test.go TestMatchDoubleStarPattern.
func TestMatchPattern_FormerJavaScriptCases(t *testing.T) {
	cases := []struct {
		name, path, pattern string
		want                bool
	}{
		{"vendor all", "vendor/pkg/file.go", "vendor/**", true},
		{"vendor nested", "vendor/a/b/c/file.go", "vendor/**", true},
		{"not vendor", "src/vendor/file.go", "vendor/**", false},
		{"node_modules root", "node_modules/react/index.js", "**/node_modules/**", true},
		{"node_modules nested", "src/node_modules/lodash/index.js", "**/node_modules/**", true},
		{"node_modules deep", "a/b/c/node_modules/pkg/file.js", "**/node_modules/**", true},
		{"not node_modules", "src/modules/file.js", "**/node_modules/**", false},
		{"node_modules exact", "node_modules", "**/node_modules/**", true},
		{"pycache", "__pycache__/file.pyc", "**/__pycache__/**", true},
		{"pycache nested", "src/__pycache__/module.pyc", "**/__pycache__/**", true},
		{"dist folder", "dist/bundle.js", "**/dist/**", true},
		{"build folder", "build/output.js", "**/build/**", true},
		{"test file", "src/utils.test.ts", "**/*.test.ts", true},
		{"spec file", "src/utils.spec.js", "**/*.spec.js", true},
		{"not test", "src/utils.ts", "**/*.test.ts", false},
		{"testdata", "testdata/sample.json", "**/testdata/**", true},
		{"testdata nested", "pkg/testdata/fixtures/data.json", "**/testdata/**", true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			assert.Equal(t, c.want, MatchPattern(c.path, c.pattern))
		})
	}
}

// #83: substring false positives (Py/JS/Kt) and multi-segment / bare-name misses (Go).
func TestMatchPattern_Issue83(t *testing.T) {
	cases := []struct {
		path, pattern string
		want          bool
	}{
		{"rebuild/app.js", "**/build/**", false},
		{"myconftest.py", "**/conftest.py", false},
		{"vendorized/x.go", "vendor/**", false},
		{"src/layout/Header.js", "**/out/**", false},
		{"pages/checkout/index.js", "**/out/**", false},
		{"pkg/unittests/x.py", "**/tests/**", false},
		{"myenv/x.py", "**/env/**", false},
		{"app/src/test/Foo.kt", "**/src/test/**", true},
		{"src/test/FooTest.kt", "**/src/test/**", true},
		{"app/mysrc/test/Foo.kt", "**/src/test/**", false},
		{"a/internal/gen/x.go", "**/internal/gen/**", true},
		{"generated", "generated", true},
		{"pkg/generated_x.go", "generated", false},
		{"x.pb.go", "*.pb.go", true},
		{"a/b/x.pb.go", "*.pb.go", true},
		{"a/b/c.go", "a/*/c.go", true},
		{"a/x/y/c.go", "a/*/c.go", false},
		{"a/x/y/c.go", "a/**/c.go", true},
		{"a/c.go", "a/**/c.go", true},
		{`pkg\vendor\x.go`, "**/vendor/**", true}, // Windows separators normalized
	}
	for _, c := range cases {
		assert.Equal(t, c.want, MatchPattern(filepath.FromSlash(c.path), c.pattern), "MatchPattern(%q, %q)", c.path, c.pattern)
	}
}

// Every language's default exclude pattern: one path it must exclude and one
// look-alike it must not. Keep in sync with each provider's DefaultExcludePatterns.
func TestMatchPattern_LanguageDefaults(t *testing.T) {
	cases := []struct{ pattern, excluded, kept string }{
		// Go
		{"vendor/**", "vendor/pkg/a.go", "vendorized/a.go"},
		{"**/*_test.go", "pkg/a_test.go", "pkg/a_test_helper.go"},
		{"**/testdata/**", "pkg/testdata/a.go", "pkg/mytestdata/a.go"},
		{"**/*.pb.go", "api/x.pb.go", "api/xpb.go"},
		// Python
		{"**/__pycache__/**", "a/__pycache__/m.pyc", "a/my__pycache__/m.py"},
		{"**/venv/**", "venv/lib/x.py", "myvenv/x.py"},
		{"**/.venv/**", ".venv/lib/x.py", "a.venv/x.py"},
		{"**/env/**", "env/x.py", "myenv/x.py"},
		{"**/.env/**", ".env/x.py", "my.env/x.py"},
		{"**/site-packages/**", "lib/site-packages/x.py", "lib/my-site-packages/x.py"},
		{"**/test_*.py", "pkg/test_a.py", "pkg/mytest_a.py"},
		{"**/*_test.py", "pkg/a_test.py", "pkg/a_test_util.py"},
		{"**/conftest.py", "tests/conftest.py", "myconftest.py"},
		{"**/tests/**", "pkg/tests/a.py", "pkg/unittests/a.py"},
		{"**/.tox/**", ".tox/x.py", "my.tox/x.py"},
		{"**/.pytest_cache/**", ".pytest_cache/x", "my.pytest_cache/x"},
		// JavaScript / TypeScript
		{"**/node_modules/**", "node_modules/r/i.js", "my_node_modules/i.js"},
		{"**/dist/**", "dist/b.js", "redist/b.js"},
		{"**/build/**", "build/o.js", "rebuild/o.js"},
		{"**/.next/**", ".next/x.js", "my.next/x.js"},
		{"**/out/**", "out/x.js", "src/layout/x.js"},
		{"**/.nuxt/**", ".nuxt/x.js", "a.nuxt/x.js"},
		{"**/.output/**", ".output/x.js", "my.output/x.js"},
		{"**/*.test.js", "src/a.test.js", "src/test.js"},
		{"**/*.test.ts", "src/a.test.ts", "src/test.ts"},
		{"**/*.test.jsx", "src/a.test.jsx", "src/test.jsx"},
		{"**/*.test.tsx", "src/a.test.tsx", "src/test.tsx"},
		{"**/*.spec.js", "src/a.spec.js", "src/spec.js"},
		{"**/*.spec.ts", "src/a.spec.ts", "src/spec.ts"},
		{"**/*.spec.jsx", "src/a.spec.jsx", "src/spec.jsx"},
		{"**/*.spec.tsx", "src/a.spec.tsx", "src/spec.tsx"},
		{"**/__tests__/**", "src/__tests__/a.js", "src/my__tests__/a.js"},
		{"**/__mocks__/**", "src/__mocks__/a.js", "src/my__mocks__/a.js"},
		{"**/*.d.ts", "types/a.d.ts", "types/a.ts"},
		{"**/*.min.js", "lib/a.min.js", "lib/a.js"},
		{"**/*.bundle.js", "lib/a.bundle.js", "lib/a.js"},
		{"**/jest.config.js", "jest.config.js", "myjest.config.js"},
		{"**/jest.config.ts", "jest.config.ts", "myjest.config.ts"},
		{"**/webpack.config.js", "webpack.config.js", "mywebpack.config.js"},
		{"**/vite.config.ts", "vite.config.ts", "myvite.config.ts"},
		{"**/vite.config.js", "vite.config.js", "myvite.config.js"},
		{"**/rollup.config.js", "rollup.config.js", "myrollup.config.js"},
		{"**/next.config.js", "next.config.js", "mynext.config.js"},
		{"**/next.config.mjs", "next.config.mjs", "mynext.config.mjs"},
		// Kotlin
		{"**/build/**", "app/build/x.kt", "app/rebuild/x.kt"},
		{"**/.gradle/**", ".gradle/x", "my.gradle/x"},
		{"**/generated/**", "a/generated/x.kt", "a/notgenerated/x.kt"},
		{"**/src/test/**", "app/src/test/F.kt", "app/src/testing/F.kt"},
		{"**/src/testFixtures/**", "app/src/testFixtures/F.kt", "app/src/testFixturesX/F.kt"},
		{"**/*Test.kt", "src/FooTest.kt", "src/FooTests.kt"},
		{"**/*Spec.kt", "src/FooSpec.kt", "src/FooSpecs.kt"},
	}
	for _, c := range cases {
		assert.True(t, MatchPattern(c.excluded, c.pattern), "%q should exclude %q", c.pattern, c.excluded)
		assert.False(t, MatchPattern(c.kept, c.pattern), "%q should keep %q", c.pattern, c.kept)
	}
}

func TestShouldExclude_ChecksAncestors(t *testing.T) {
	root := filepath.FromSlash("/repo/project")
	join := func(rel string) string { return filepath.Join(root, filepath.FromSlash(rel)) }

	// A file list has no walk to prune directories, so ancestors must be checked.
	assert.True(t, ShouldExclude(join("node_modules/react/lib/index.js"), root, []string{"node_modules"}))
	assert.True(t, ShouldExclude(join("pkg/generated/deep/x.go"), root, []string{"generated"}))
	assert.True(t, ShouldExclude(join("vendor/a/b.go"), root, []string{"vendor/**"}))
	assert.False(t, ShouldExclude(join("src/app.go"), root, []string{"node_modules", "vendor/**"}))

	// The root itself is never excluded, even if its name matches a pattern.
	assert.False(t, ShouldExclude(filepath.FromSlash("/repo/build"), filepath.FromSlash("/repo/build"), []string{"build"}))
	// Relative roots (the CLI default is ".").
	assert.True(t, ShouldExclude(filepath.FromSlash("vendor/x.go"), ".", []string{"vendor/**"}))
	assert.False(t, ShouldExclude("main.go", ".", nil))
}

// Bare names match at any depth and anchored literals from the root; for files
// inside such a directory this happens through the ancestor check (#83).
func TestShouldExclude_BareAndAnchoredNames(t *testing.T) {
	root := filepath.FromSlash("/r")
	cases := []struct {
		path, pattern string
		want          bool
	}{
		{"pkg/generated/x.go", "generated", true},
		{"generated/x.go", "generated", true},
		{"internal/gen/x.go", "internal/gen", true},
		{"a/internal/gen/x.go", "internal/gen", false},
	}
	for _, c := range cases {
		got := ShouldExclude(filepath.Join(root, filepath.FromSlash(c.path)), root, []string{c.pattern})
		assert.Equal(t, c.want, got, "ShouldExclude(%q, %q)", c.path, c.pattern)
	}
}

func TestHasMatchingExtension(t *testing.T) {
	assert.True(t, HasMatchingExtension("a/b.go", []string{".go"}))
	assert.True(t, HasMatchingExtension("a/b.tsx", []string{".ts", ".tsx"}))
	assert.False(t, HasMatchingExtension("a/b.gox", []string{".go"}))
	assert.False(t, HasMatchingExtension("a/b.go", nil))
}
