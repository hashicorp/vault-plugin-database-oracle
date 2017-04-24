/*
Copyright 2014 Zachary Klippenstein
Copyright 2017 Graham Davison

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

/*
Package regen is a library for generating random strings from regular expressions.
The generated strings will match the expressions they were generated from. Similar
to Ruby's randexp library.

E.g.
	regen.Generate("[a-z0-9]{1,64}")
will return a lowercase alphanumeric string
between 1 and 64 characters long.

Expressions are parsed using the Go standard library's parser: http://golang.org/pkg/regexp/syntax/.

Constraints

"." will generate any character, not necessarily a printable one.

"x{0,}", "x*", and "x+" will generate a random number of x's up to an arbitrary limit.
If you care about the maximum number, specify it explicitly in the expression,
e.g. "x{0,256}".

Flags

Flags can be passed to the parser by setting them in the GeneratorArgs struct.
Newline flags are respected, and newlines won't be generated unless the appropriate flags for
matching them are set.

E.g.
Generate(".|[^a]") will never generate newlines. To generate newlines, create a generator and pass
the flag syntax.MatchNL.

The Perl character class flag is supported, and required if the pattern contains them.

Unicode groups are not supported at this time. Support may be added in the future.

Concurrent Use

A generator can safely be used from multiple goroutines without locking.

Benchmarks

Benchmarks are included for creating and running generators for limited-length,
complex regexes, and simple, highly-repetitive regexes.

	go test -bench .

The complex benchmarks generate fake HTTP messages with the following regex:
	POST (/[-a-zA-Z0-9_.]{3,12}){3,6}
	Content-Length: [0-9]{2,3}
	X-Auth-Token: [a-zA-Z0-9+/]{64}

	([A-Za-z0-9+/]{64}
	){3,15}[A-Za-z0-9+/]{60}([A-Za-z0-9+/]{2}==|[A-Za-z0-9+/]{3}=)

The repetitive benchmarks use the regex
	a{999}

See regen_benchmarks_test.go for more information.
*/
package regen

import (
	"fmt"
	"regexp/syntax"
)

// DefaultMaxUnboundedRepeatCount is default value for MaxUnboundedRepeatCount.
const DefaultMaxUnboundedRepeatCount = 4096

// CaptureGroupHandler is a function that is called for each capture group in a regular expression.
// index and name are the index and name of the group. If unnamed, name is empty. The first capture group has index 0
// (not 1, as when matching).
// group is the regular expression within the group (e.g. for `(\w+)`, group would be `\w+`).
// generator is the generator for group.
// args is the args used to create the generator calling this function.
type CaptureGroupHandler func(index int, name string, group *syntax.Regexp, generator Generator, args *GeneratorArgs) string

// GeneratorArgs are arguments passed to NewGenerator that control how generators
// are created.
type GeneratorArgs struct {
	// Default is 0 (syntax.POSIX).
	Flags syntax.Flags

	// Maximum number of instances to generate for unbounded repeat expressions (e.g. ".*" and "{1,}")
	// Default is DefaultMaxUnboundedRepeatCount.
	MaxUnboundedRepeatCount uint
	// Minimum number of instances to generate for unbounded repeat expressions (e.g. ".*")
	// Default is 0.
	MinUnboundedRepeatCount uint

	// Set this to perform special processing of capture groups (e.g. `(\w+)`). The zero value will generate strings
	// from the expressions in the group.
	CaptureGroupHandler CaptureGroupHandler

	// maybe split this out, it's not really args
	// source of random bytes
	randomSource *rand
}

func (a *GeneratorArgs) initialize() error {
	a.randomSource = NewRand()

	// unicode groups only allowed with Perl
	if (a.Flags&syntax.UnicodeGroups) == syntax.UnicodeGroups && (a.Flags&syntax.Perl) != syntax.Perl {
		return generatorError(nil, "UnicodeGroups not supported")
	}

	if a.MaxUnboundedRepeatCount < 1 {
		a.MaxUnboundedRepeatCount = DefaultMaxUnboundedRepeatCount
	}

	if a.MinUnboundedRepeatCount > a.MaxUnboundedRepeatCount {
		panic(fmt.Sprintf("MinUnboundedRepeatCount(%d) > MaxUnboundedRepeatCount(%d)",
			a.MinUnboundedRepeatCount, a.MaxUnboundedRepeatCount))
	}

	if a.CaptureGroupHandler == nil {
		a.CaptureGroupHandler = defaultCaptureGroupHandler
	}

	return nil
}

//// Rng returns the random number generator used by generators.
//// Panics if called before the GeneratorArgs has been initialized by NewGenerator.
//func (a *GeneratorArgs) Rng() *mrand.Rand {
//	if a.rng == nil {
//		panic("GeneratorArgs has not been initialized by NewGenerator yet")
//	}
//	return a.rng
//}

// Generator generates random strings.
type Generator interface {
	Generate() string
	String() string
}

/*
Generate a random string that matches the regular expression pattern.
If args is nil, default values are used.

This function does not seed the default RNG, so you must call rand.Seed() if you want
non-deterministic strings.
*/
func Generate(pattern string) (string, error) {
	generator, err := NewGenerator(pattern, nil)
	if err != nil {
		return "", err
	}
	return generator.Generate(), nil
}

// NewGenerator creates a generator that returns random strings that match the regular expression in pattern.
// If args is nil, default values are used.
func NewGenerator(pattern string, inputArgs *GeneratorArgs) (generator Generator, err error) {
	args := GeneratorArgs{}

	// Copy inputArgs so the caller can't change them.
	if inputArgs != nil {
		args = *inputArgs
	}
	if err = args.initialize(); err != nil {
		return nil, err
	}

	var regexp *syntax.Regexp
	regexp, err = syntax.Parse(pattern, args.Flags)
	if err != nil {
		return
	}

	var gen *internalGenerator
	gen, err = newGenerator(regexp, &args)
	if err != nil {
		return
	}

	return gen, nil
}
