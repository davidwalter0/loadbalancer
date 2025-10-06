// originally: https://golang.org/src/flag/flag.go

// 2017 Copyright David Walter
// Released under the same BSD-style licenses mentioned below.

// modified to add more flag types in support of types in config

// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

/*
	Package flag implements command-line flag parsing.

	Usage:

	Define flags using flag.String(), Bool(), Int(), etc.

  Note: deprecate using of flags.Parse in init() methods

	This declares an integer flag, -flagname, stored in the pointer ip, with type *int.
		import "flag"
		var ip = flag.Int("flagname", 1234, "help message for flagname")
	If you like, you can bind the flag to a variable using the Var() functions.
		var flagvar int
  	flag.IntVar(&flagvar, "flagname", 1234, "help message for flagname")
	Or you can create custom flags that satisfy the Value interface (with
	pointer receivers) and couple them to flag parsing by
		flag.Var(&flagVal, "name", "help message for flagname")
	For such flags, the default value is just the initial value of the variable.

	After all flags are defined, call
		flag.Parse()
	to parse the command line into the defined flags.

	Flags may then be used directly. If you're using the flags themselves,
	they are all pointers; if you bind to variables, they're values.
		fmt.Println("ip has value ", *ip)
		fmt.Println("flagvar has value ", flagvar)

	After parsing, the arguments following the flags are available as the
	slice flag.Args() or individually as flag.Arg(i).
	The arguments are indexed from 0 through flag.NArg()-1.

	Command line flag syntax:
		-flag
		-flag=x
		-flag x  // non-boolean flags only
	One or two minus signs may be used; they are equivalent.
	The last form is not permitted for boolean flags because the
	meaning of the command
		cmd -x *
	will change if there is a file called 0, false, etc.  You must
	use the -flag=false form to turn off a boolean flag.

	Flag parsing stops just before the first non-flag argument
	("-" is a non-flag argument) or after the terminator "--".

	Integer flags accept 1234, 0664, 0x1234 and may be negative.
	Boolean flags may be:
		1, 0, t, f, T, F, true, false, TRUE, FALSE, True, False
	Duration flags accept any input valid for time.ParseDuration.

	The default set of command-line flags is controlled by
	top-level functions.  The FlagSet type allows one to define
	independent sets of flags, such as to implement subcommands
	in a command-line interface. The methods of FlagSet are
	analogous to the top-level functions for the command-line
	flag set.
*/

package eflag // import "github.com/davidwalter0/go-flag"

import (
	"errors"
	"fmt"
	"io"
	"os"
	"reflect"
	"sort"
	"strconv"

	"time"
)

// ErrHelp is the error returned if the -help or -h flag is invoked
// but no such flag is defined.
var ErrHelp = errors.New("flag: help requested")

// -- bool Value
type boolValue bool

func newBoolValue(val bool, p *bool) *boolValue {
	*p = val
	return (*boolValue)(p)
}

func (b *boolValue) Set(s string) error {
	v, err := strconv.ParseBool(s)
	*b = boolValue(v)
	return err
}

func (b *boolValue) Get() interface{} { return bool(*b) }

func (b *boolValue) String() string { return strconv.FormatBool(bool(*b)) }

func (b *boolValue) IsBoolFlag() bool { return true }

// optional interface to indicate boolean flags that can be
// supplied without "=value" text
type boolFlag interface {
	Value
	IsBoolFlag() bool
}

// -- int Value
type intValue int

func newIntValue(val int, p *int) *intValue {
	*p = val
	return (*intValue)(p)
}

func (i *intValue) Set(s string) (err error) {
	if len(s) > 0 {
		var v int64
		v, err = strconv.ParseInt(s, 0, 32)
		*i = intValue(v)
	}
	return
}

func (i *intValue) Get() interface{} { return int(*i) }

func (i *intValue) String() string { return strconv.Itoa(int(*i)) }

// -- int64 Value
type int64Value int64

func newInt64Value(val int64, p *int64) *int64Value {
	*p = val
	return (*int64Value)(p)
}

func (i *int64Value) Set(s string) (err error) {
	if len(s) > 0 {
		var v int64
		v, err = strconv.ParseInt(s, 0, 64)
		*i = int64Value(v)
	}
	return
}

func (i *int64Value) Get() interface{} { return int64(*i) }

func (i *int64Value) String() string { return strconv.FormatInt(int64(*i), 10) }

// -- uint Value
type uintValue uint

func newUintValue(val uint, p *uint) *uintValue {
	*p = val
	return (*uintValue)(p)
}

func (i *uintValue) Set(s string) (err error) {
	if len(s) > 0 {
		var v uint64
		v, err = strconv.ParseUint(s, 0, 32)
		*i = uintValue(v)
	}
	return
}

func (i *uintValue) Get() interface{} { return uint(*i) }

func (i *uintValue) String() string { return strconv.FormatUint(uint64(*i), 10) }

// -- uint64 Value
type uint64Value uint64

func newUint64Value(val uint64, p *uint64) *uint64Value {
	*p = val
	return (*uint64Value)(p)
}

func (i *uint64Value) Set(s string) (err error) {
	if len(s) > 0 {
		var v uint64
		v, err = strconv.ParseUint(s, 0, 64)
		*i = uint64Value(v)
	}
	return err
}

func (i *uint64Value) Get() interface{} { return uint64(*i) }

func (i *uint64Value) String() string { return strconv.FormatUint(uint64(*i), 10) }

// -- string Value
type stringValue string

func newStringValue(val string, p *string) *stringValue {
	*p = val
	return (*stringValue)(p)
}

func (s *stringValue) Set(val string) error {
	*s = stringValue(val)
	return nil
}

func (s *stringValue) Get() interface{} { return string(*s) }

func (s *stringValue) String() string { return string(*s) }

// -- float32 Value
type float32Value float32

func newFloat32Value(val float32, p *float32) *float32Value {
	*p = val
	return (*float32Value)(p)
}

func (f *float32Value) Set(s string) (err error) {
	if len(s) > 0 {
		var v float64
		v, err = strconv.ParseFloat(s, 32)
		*f = float32Value(v)
	}
	return err
}

func (f *float32Value) Get() interface{} { return float32(*f) }

func (f *float32Value) String() string { return strconv.FormatFloat(float64(*f), 'g', -1, 32) }

// -- float64 Value
type float64Value float64

func newFloat64Value(val float64, p *float64) *float64Value {
	*p = val
	return (*float64Value)(p)
}

func (f *float64Value) Set(s string) (err error) {
	if len(s) > 0 {
		var v float64
		v, err = strconv.ParseFloat(s, 64)
		*f = float64Value(v)
	}
	return err
}

func (f *float64Value) Get() interface{} { return float64(*f) }

func (f *float64Value) String() string { return strconv.FormatFloat(float64(*f), 'g', -1, 64) }

// -- time.Duration Value
type durationValue time.Duration

func newDurationValue(val time.Duration, p *time.Duration) *durationValue {
	*p = val
	return (*durationValue)(p)
}

func (d *durationValue) Set(s string) (err error) {
	if len(s) > 0 {
		var v time.Duration
		v, err = time.ParseDuration(s)
		*d = durationValue(v)
	}
	return err
}

func (d *durationValue) Get() interface{} { return time.Duration(*d) }

func (d *durationValue) String() string { return (*time.Duration)(d).String() }

// Value is the interface to the dynamic value stored in a flag.
// (The default value is represented as a string.)
//
// If a Value has an IsBoolFlag() bool method returning true,
// the command-line parser makes -name equivalent to -name=true
// rather than using the next command-line argument.
//
// Set is called once, in command line order, for each flag present.
// The flag package may call the String method with a zero-valued receiver,
// such as a nil pointer.
type Value interface {
	String() string
	Set(string) error
}

// Getter is an interface that allows the contents of a Value to be retrieved.
// It wraps the Value interface, rather than being part of it, because it
// appeared after Go 1 and its compatibility rules. All Value types provided
// by this package satisfy the Getter interface.
type Getter interface {
	Value
	Get() interface{}
}

// ErrorHandling defines how FlagSet.Parse behaves if the parse fails.
type ErrorHandling int

// These constants cause FlagSet.Parse to behave as described if the parse fails.
const (
	ContinueOnError ErrorHandling = iota // Return a descriptive error.
	ExitOnError                          // Call os.Exit(2).
	PanicOnError                         // Call panic with a descriptive error.
)

// A FlagSet represents a set of defined flags. The zero value of a FlagSet
// has no name and has ContinueOnError error handling.
type FlagSet struct {
	// Usage is the function called when an error occurs while parsing flags.
	// The field is a function (not a method) that may be changed to point to
	// a custom error handler.
	Usage func()

	name          string
	parsed        bool
	actual        map[string]*Flag
	formal        map[string]*Flag
	args          []string // arguments after flags
	errorHandling ErrorHandling
	output        io.Writer // nil means stderr; use out() accessor
}

func (f *FlagSet) Reset() {
	f.Usage = func() {}
	f.parsed = false
	f.actual = nil
	f.formal = nil
	f.args = nil
	f.errorHandling = ContinueOnError
	f.output = nil
}

// A Flag represents the state of a flag.
type Flag struct {
	Name     string // name as it appears on command line
	Usage    string // help message
	Value    Value  // value as set
	DefValue string // default value (as text); for usage message
	Required bool   // must set the value
	IsSet    bool   // has been altered
}

// sortFlags returns the flags as a slice in lexicographical sorted order.
func sortFlags(flags map[string]*Flag) []*Flag {
	list := make(sort.StringSlice, len(flags))
	i := 0
	for _, f := range flags {
		list[i] = f.Name
		i++
	}
	list.Sort()
	result := make([]*Flag, len(list))
	for i, name := range list {
		result[i] = flags[name]
	}
	return result
}

func (f *FlagSet) out() io.Writer {
	if f.output == nil {
		return os.Stderr
	}
	return f.output
}

// SetOutput sets the destination for usage and error messages.
// If output is nil, os.Stderr is used.
func (f *FlagSet) SetOutput(output io.Writer) {
	f.output = output
}

// VisitAll visits the flags in lexicographical order, calling fn for each.
// It visits all flags, even those not set.
func (f *FlagSet) VisitAll(fn func(*Flag)) {
	for _, flag := range sortFlags(f.formal) {
		fn(flag)
	}
}

// VisitAll visits the command-line flags in lexicographical order, calling
// fn for each. It visits all flags, even those not set.
func VisitAll(fn func(*Flag)) {
	CommandLine.VisitAll(fn)
}

// Visit visits the flags in lexicographical order, calling fn for each.
// It visits only those flags that have been set.
func (f *FlagSet) Visit(fn func(*Flag)) {
	for _, flag := range sortFlags(f.actual) {
		fn(flag)
	}
}

// Visit visits the command-line flags in lexicographical order, calling fn
// for each. It visits only those flags that have been set.
func Visit(fn func(*Flag)) {
	CommandLine.Visit(fn)
}

// Lookup returns the Flag structure of the named flag, returning nil if none exists.
func (f *FlagSet) Lookup(name string) *Flag {
	return f.formal[name]
}

// Lookup returns the Flag structure of the named command-line flag,
// returning nil if none exists.
func Lookup(name string) *Flag {
	return CommandLine.formal[name]
}

// Set sets the value of the named flag.
func (f *FlagSet) Set(name, value string) error {
	flag, ok := f.formal[name]
	if !ok {
		return fmt.Errorf("no such flag -%v", name)
	}
	err := flag.Value.Set(value)
	if err != nil {
		return err
	}
	if f.actual == nil {
		f.actual = make(map[string]*Flag)
	}
	f.actual[name] = flag
	return nil
}

// Set sets the value of the named command-line flag.
func Set(name, value string) error {
	return CommandLine.Set(name, value)
}

// isZeroValue guesses whether the string represents the zero
// value for a flag. It is not accurate but in practice works OK.
func isZeroValue(flag *Flag, value string) bool {
	// Build a zero value of the flag's Value type, and see if the
	// result of calling its String method equals the value passed in.
	// This works unless the Value type is itself an interface type.
	typ := reflect.TypeOf(flag.Value)
	var z reflect.Value
	if typ.Kind() == reflect.Ptr {
		z = reflect.New(typ.Elem())
	} else {
		z = reflect.Zero(typ)
	}
	if value == z.Interface().(Value).String() {
		return true
	}

	switch value {
	case "false":
		return true
	case "":
		return true
	case "0":
		return true
	}
	return false
}

// UnquoteUsage extracts a back-quoted name from the usage
// string for a flag and returns it and the un-quoted usage.
// Given "a `name` to show" it returns ("name", "a name to show").
// If there are no back quotes, the name is an educated guess of the
// type of the flag's value, or the empty string if the flag is boolean.
func UnquoteUsage(flag *Flag) (name string, usage string) {
	// Look for a back-quoted name, but avoid the strings package.
	usage = flag.Usage
	for i := 0; i < len(usage); i++ {
		if usage[i] == '`' {
			for j := i + 1; j < len(usage); j++ {
				if usage[j] == '`' {
					name = usage[i+1 : j]
					usage = usage[:i] + name + usage[j+1:]
					return name, usage
				}
			}
			break // Only one back quote; use type name.
		}
	}
	// No explicit name, so use type if we can find one.
	name = "value"
	switch flag.Value.(type) {
	case boolFlag:
		name = ""
	case *durationValue:
		name = "duration"
	case *float64Value:
		name = "float"
	case *float32Value:
		name = "float32"
	case *intValue, *int64Value:
		name = "int"
	case *stringValue:
		name = "string"
	case *uintValue, *uint64Value:
		name = "uint"
	case *uint8Value:
		name = "uint8"
	case *uint16Value:
		name = "uint16"
	case *uint32Value:
		name = "uint32"
	case *int8Value:
		name = "int8"
	case *int16Value:
		name = "int16"
	case *int32Value:
		name = "int32"
	}
	return
}

// PrintDefaults prints to standard error the default values of all
// defined command-line flags in the set. See the documentation for
// the global function PrintDefaults for more information.
func (f *FlagSet) PrintDefaults() {
	f.VisitAll(func(flag *Flag) {
		var s string
		if !SupressFlagUsageMsg {
			s = fmt.Sprintf("  -%s", flag.Name) // Two spaces before -; see next two comments.
		} else {
			s += "  "
		}
		// name is type name, usage is flag usage argument
		name, usage := UnquoteUsage(flag)
		if len(name) > 0 && !SupressFlagUsageMsg {
			s += " " + name
		}
		// Boolean flags of one ASCII letter are so common we
		// treat them specially, putting their usage on the same line.
		if len(s) <= 4 { // space, space, '-', 'x'.
			if !SupressFlagUsageMsg {
				s += "\t"
			}
		} else {
			// Four spaces before the tab triggers good alignment
			// for both 4- and 8-space tab stops.
			if !SupressFlagUsageMsg {
				s += "\n    \t"
			}
		}
		s += usage
		if !isZeroValue(flag, flag.DefValue) {
			if _, ok := flag.Value.(*stringValue); ok {
				// put quotes on the value
				s += fmt.Sprintf(" (default %q)", flag.DefValue)
			} else {
				s += fmt.Sprintf(" (default %v)", flag.DefValue)
			}
		}
		fmt.Fprint(f.out(), s, "\n")
	})
}

// PrintDefaults prints, to standard error unless configured otherwise,
// a usage message showing the default settings of all defined
// command-line flags.
// For an integer valued flag x, the default output has the form
//	-x int
//		usage-message-for-x (default 7)
// The usage message will appear on a separate line for anything but
// a bool flag with a one-byte name. For bool flags, the type is
// omitted and if the flag name is one byte the usage message appears
// on the same line. The parenthetical default is omitted if the
// default is the zero value for the type. The listed type, here int,
// can be changed by placing a back-quoted name in the flag's usage
// string; the first such item in the message is taken to be a parameter
// name to show in the message and the back quotes are stripped from
// the message when displayed. For instance, given
//	flag.String("I", "", "search `directory` for include files")
// the output will be
//	-I directory
//		search directory for include files.
func PrintDefaults() {
	CommandLine.PrintDefaults()
}

// defaultUsage is the default function to print a usage message.
func (f *FlagSet) defaultUsage() {
	if f.name == "" {
		fmt.Fprintf(f.out(), "Usage:\n")
	} else {
		fmt.Fprintf(f.out(), "Usage of %s:\n", f.name)
	}
	f.PrintDefaults()
}

// NOTE: Usage is not just defaultUsage(CommandLine)
// because it serves (via godoc flag Usage) as the example
// for how to write your own usage function.

// Usage prints to standard error a usage message documenting all defined command-line flags.
// It is called when an error occurs while parsing flags.
// The function is a variable that may be changed to point to a custom function.
// By default it prints a simple header and calls PrintDefaults; for details about the
// format of the output and how to control it, see the documentation for PrintDefaults.
var Usage = func() {
	fmt.Fprintf(os.Stderr, "Usage of %s:\n", os.Args[0])
	PrintDefaults()
}

// NFlag returns the number of flags that have been set.
func (f *FlagSet) NFlag() int { return len(f.actual) }

// IsSet returns whether the named flag was set
func (f *FlagSet) IsSet(name string) bool {
	_, ok := f.actual[name]
	v, found := f.formal[name]
	return ok || (found && v.IsSet)
}

// Required returns whether the named flag was set
func (f *FlagSet) Required(name string) bool {
	v, found := f.formal[name]
	return found && v.Required
}

// Ok returns whether the named flag was set
func (f *FlagSet) Ok(name string) bool {
	return !f.Required(name) || f.IsSet(name)
}

// NFlag returns the number of command-line flags that have been set.
func NFlag() int { return len(CommandLine.actual) }

// IsSet returns whether the named flag was set
func IsSet(name string) bool {
	_, ok := CommandLine.actual[name]
	v, found := CommandLine.formal[name]
	return ok || (found && v.IsSet)
}

// Required returns whether the named flag was set
func Required(name string) bool {
	v, found := CommandLine.formal[name]
	return found && v.Required
}

// Ok returns whether the named flag was set
func Ok(name string) bool {
	return !Required(name) || IsSet(name)
}

// Arg returns the i'th argument. Arg(0) is the first remaining argument
// after flags have been processed. Arg returns an empty string if the
// requested element does not exist.
func (f *FlagSet) Arg(i int) string {
	if i < 0 || i >= len(f.args) {
		return ""
	}
	return f.args[i]
}

// Arg returns the i'th command-line argument. Arg(0) is the first remaining argument
// after flags have been processed. Arg returns an empty string if the
// requested element does not exist.
func Arg(i int) string {
	return CommandLine.Arg(i)
}

// NArg is the number of arguments remaining after flags have been processed.
func (f *FlagSet) NArg() int { return len(f.args) }

// NArg is the number of arguments remaining after flags have been processed.
func NArg() int { return len(CommandLine.args) }

// Args returns the non-flag arguments.
func (f *FlagSet) Args() []string { return f.args }

// Args returns the non-flag command-line arguments.
func Args() []string { return CommandLine.args }

// BoolVar defines a bool flag with specified name, default value, and usage string.
// The argument p points to a bool variable in which to store the value of the flag.
func (f *FlagSet) BoolVar(p *bool, name string, value bool, usage string, required, isset bool) {
	f.Var(newBoolValue(value, p), name, usage, required, isset)
}

// BoolVar defines a bool flag with specified name, default value, and usage string.
// The argument p points to a bool variable in which to store the value of the flag.
func BoolVar(p *bool, name string, value bool, usage string, required, isset bool) {
	CommandLine.Var(newBoolValue(value, p), name, usage, required, isset)
}

// Bool defines a bool flag with specified name, default value, and usage string.
// The return value is the address of a bool variable that stores the value of the flag.
func (f *FlagSet) Bool(name string, value bool, usage string, required, isset bool) *bool {
	p := new(bool)
	f.BoolVar(p, name, value, usage, required, isset)
	return p
}

// Bool defines a bool flag with specified name, default value, and usage string.
// The return value is the address of a bool variable that stores the value of the flag.
func Bool(name string, value bool, usage string, required, isset bool) *bool {
	return CommandLine.Bool(name, value, usage, required, isset)
}

// IntVar defines an int flag with specified name, default value, and usage string.
// The argument p points to an int variable in which to store the value of the flag.
func (f *FlagSet) IntVar(p *int, name string, value int, usage string, required, isset bool) {
	f.Var(newIntValue(value, p), name, usage, required, isset)
}

// IntVar defines an int flag with specified name, default value, and usage string.
// The argument p points to an int variable in which to store the value of the flag.
func IntVar(p *int, name string, value int, usage string, required, isset bool) {
	CommandLine.Var(newIntValue(value, p), name, usage, required, isset)
}

// Int defines an int flag with specified name, default value, and usage string.
// The return value is the address of an int variable that stores the value of the flag.
func (f *FlagSet) Int(name string, value int, usage string, required, isset bool) *int {
	p := new(int)
	f.IntVar(p, name, value, usage, required, isset)
	return p
}

// Int defines an int flag with specified name, default value, and usage string.
// The return value is the address of an int variable that stores the value of the flag.
func Int(name string, value int, usage string, required, isset bool) *int {
	return CommandLine.Int(name, value, usage, required, isset)
}

// Int64Var defines an int64 flag with specified name, default value, and usage string.
// The argument p points to an int64 variable in which to store the value of the flag.
func (f *FlagSet) Int64Var(p *int64, name string, value int64, usage string, required, isset bool) {
	f.Var(newInt64Value(value, p), name, usage, required, isset)
}

// Int64Var defines an int64 flag with specified name, default value, and usage string.
// The argument p points to an int64 variable in which to store the value of the flag.
func Int64Var(p *int64, name string, value int64, usage string, required, isset bool) {
	CommandLine.Var(newInt64Value(value, p), name, usage, required, isset)
}

// Int64 defines an int64 flag with specified name, default value, and usage string.
// The return value is the address of an int64 variable that stores the value of the flag.
func (f *FlagSet) Int64(name string, value int64, usage string, required, isset bool) *int64 {
	p := new(int64)
	f.Int64Var(p, name, value, usage, required, isset)
	return p
}

// Int64 defines an int64 flag with specified name, default value, and usage string.
// The return value is the address of an int64 variable that stores the value of the flag.
func Int64(name string, value int64, usage string, required, isset bool) *int64 {
	return CommandLine.Int64(name, value, usage, required, isset)
}

// UintVar defines a uint flag with specified name, default value, and usage string.
// The argument p points to a uint variable in which to store the value of the flag.
func (f *FlagSet) UintVar(p *uint, name string, value uint, usage string, required, isset bool) {
	f.Var(newUintValue(value, p), name, usage, required, isset)
}

// UintVar defines a uint flag with specified name, default value, and usage string.
// The argument p points to a uint  variable in which to store the value of the flag.
func UintVar(p *uint, name string, value uint, usage string, required, isset bool) {
	CommandLine.Var(newUintValue(value, p), name, usage, required, isset)
}

// Uint defines a uint flag with specified name, default value, and usage string.
// The return value is the address of a uint  variable that stores the value of the flag.
func (f *FlagSet) Uint(name string, value uint, usage string, required, isset bool) *uint {
	p := new(uint)
	f.UintVar(p, name, value, usage, required, isset)
	return p
}

// Uint defines a uint flag with specified name, default value, and usage string.
// The return value is the address of a uint  variable that stores the value of the flag.
func Uint(name string, value uint, usage string, required, isset bool) *uint {
	return CommandLine.Uint(name, value, usage, required, isset)
}

// Uint64Var defines a uint64 flag with specified name, default value, and usage string.
// The argument p points to a uint64 variable in which to store the value of the flag.
func (f *FlagSet) Uint64Var(p *uint64, name string, value uint64, usage string, required, isset bool) {
	f.Var(newUint64Value(value, p), name, usage, required, isset)
}

// Uint64Var defines a uint64 flag with specified name, default value, and usage string.
// The argument p points to a uint64 variable in which to store the value of the flag.
func Uint64Var(p *uint64, name string, value uint64, usage string, required, isset bool) {
	CommandLine.Var(newUint64Value(value, p), name, usage, required, isset)
}

// Uint64 defines a uint64 flag with specified name, default value, and usage string.
// The return value is the address of a uint64 variable that stores the value of the flag.
func (f *FlagSet) Uint64(name string, value uint64, usage string, required, isset bool) *uint64 {
	p := new(uint64)
	f.Uint64Var(p, name, value, usage, required, isset)
	return p
}

// Uint64 defines a uint64 flag with specified name, default value, and usage string.
// The return value is the address of a uint64 variable that stores the value of the flag.
func Uint64(name string, value uint64, usage string, required, isset bool) *uint64 {
	return CommandLine.Uint64(name, value, usage, required, isset)
}

// StringVar defines a string flag with specified name, default value, and usage string.
// The argument p points to a string variable in which to store the value of the flag.
func (f *FlagSet) StringVar(p *string, name string, value string, usage string, required, isset bool) {
	f.Var(newStringValue(value, p), name, usage, required, isset)
}

// StringVar defines a string flag with specified name, default value, and usage string.
// The argument p points to a string variable in which to store the value of the flag.
func StringVar(p *string, name string, value string, usage string, required, isset bool) {
	CommandLine.Var(newStringValue(value, p), name, usage, required, isset)
}

// String defines a string flag with specified name, default value, and usage string.
// The return value is the address of a string variable that stores the value of the flag.
func (f *FlagSet) String(name string, value string, usage string, required, isset bool) *string {
	p := new(string)
	f.StringVar(p, name, value, usage, required, isset)
	return p
}

// String defines a string flag with specified name, default value, and usage string.
// The return value is the address of a string variable that stores the value of the flag.
func String(name string, value string, usage string, required, isset bool) *string {
	return CommandLine.String(name, value, usage, required, isset)
}

// Float32Var defines a float32 flag with specified name, default value, and usage string.
// The argument p points to a float32 variable in which to store the value of the flag.
func (f *FlagSet) Float32Var(p *float32, name string, value float32, usage string, required, isset bool) {
	f.Var(newFloat32Value(value, p), name, usage, required, isset)
}

// Float32Var defines a float32 flag with specified name, default value, and usage string.
// The argument p points to a float32 variable in which to store the value of the flag.
func Float32Var(p *float32, name string, value float32, usage string, required, isset bool) {
	CommandLine.Var(newFloat32Value(value, p), name, usage, required, isset)
}

// Float32 defines a float32 flag with specified name, default value, and usage string.
// The return value is the address of a float32 variable that stores the value of the flag.
func (f *FlagSet) Float32(name string, value float32, usage string, required, isset bool) *float32 {
	p := new(float32)
	f.Float32Var(p, name, value, usage, required, isset)
	return p
}

// Float32 defines a float32 flag with specified name, default value, and usage string.
// The return value is the address of a float32 variable that stores the value of the flag.
func Float32(name string, value float32, usage string, required, isset bool) *float32 {
	return CommandLine.Float32(name, value, usage, required, isset)
}

// Float64Var defines a float64 flag with specified name, default value, and usage string.
// The argument p points to a float64 variable in which to store the value of the flag.
func (f *FlagSet) Float64Var(p *float64, name string, value float64, usage string, required, isset bool) {
	f.Var(newFloat64Value(value, p), name, usage, required, isset)
}

// Float64Var defines a float64 flag with specified name, default value, and usage string.
// The argument p points to a float64 variable in which to store the value of the flag.
func Float64Var(p *float64, name string, value float64, usage string, required, isset bool) {
	CommandLine.Var(newFloat64Value(value, p), name, usage, required, isset)
}

// Float64 defines a float64 flag with specified name, default value, and usage string.
// The return value is the address of a float64 variable that stores the value of the flag.
func (f *FlagSet) Float64(name string, value float64, usage string, required, isset bool) *float64 {
	p := new(float64)
	f.Float64Var(p, name, value, usage, required, isset)
	return p
}

// Float64 defines a float64 flag with specified name, default value, and usage string.
// The return value is the address of a float64 variable that stores the value of the flag.
func Float64(name string, value float64, usage string, required, isset bool) *float64 {
	return CommandLine.Float64(name, value, usage, required, isset)
}

// DurationVar defines a time.Duration flag with specified name, default value, and usage string.
// The argument p points to a time.Duration variable in which to store the value of the flag.
// The flag accepts a value acceptable to time.ParseDuration.
func (f *FlagSet) DurationVar(p *time.Duration, name string, value time.Duration, usage string, required, isset bool) {
	f.Var(newDurationValue(value, p), name, usage, required, isset)
}

// DurationVar defines a time.Duration flag with specified name, default value, and usage string.
// The argument p points to a time.Duration variable in which to store the value of the flag.
// The flag accepts a value acceptable to time.ParseDuration.
func DurationVar(p *time.Duration, name string, value time.Duration, usage string, required, isset bool) {
	CommandLine.Var(newDurationValue(value, p), name, usage, required, isset)
}

// Duration defines a time.Duration flag with specified name, default value, and usage string.
// The return value is the address of a time.Duration variable that stores the value of the flag.
// The flag accepts a value acceptable to time.ParseDuration.
func (f *FlagSet) Duration(name string, value time.Duration, usage string, required, isset bool) *time.Duration {
	p := new(time.Duration)
	f.DurationVar(p, name, value, usage, required, isset)
	return p
}

// Duration defines a time.Duration flag with specified name, default value, and usage string.
// The return value is the address of a time.Duration variable that stores the value of the flag.
// The flag accepts a value acceptable to time.ParseDuration.
func Duration(name string, value time.Duration, usage string, required, isset bool) *time.Duration {
	return CommandLine.Duration(name, value, usage, required, isset)
}

// Var defines a flag with the specified name and usage string. The type and
// value of the flag are represented by the first argument, of type Value, which
// typically holds a user-defined implementation of Value. For instance, the
// caller could create a flag that turns a comma-separated string into a slice
// of strings by giving the slice the methods of Value; in particular, Set would
// decompose the comma-separated string into the slice.
func (f *FlagSet) Var(value Value, name, usage string, required, isset bool) {
	// Remember the default value as a string; it won't change.
	flag := &Flag{
		Name:     name,
		Usage:    usage,
		Value:    value,
		DefValue: value.String(),
		Required: required,
		IsSet:    isset,
	}
	_, alreadythere := f.formal[name]
	if alreadythere {
		var msg string
		if f.name == "" {
			msg = fmt.Sprintf("flag redefined: %s", name)
		} else {
			msg = fmt.Sprintf("%s flag redefined: %s", f.name, name)
		}
		fmt.Fprintln(f.out(), msg)
		panic(msg) // Happens only if flags are declared with identical names
	}
	if f.formal == nil {
		f.formal = make(map[string]*Flag)
	}
	f.formal[name] = flag
}

// Var defines a flag with the specified name and usage string. The type and
// value of the flag are represented by the first argument, of type Value, which
// typically holds a user-defined implementation of Value. For instance, the
// caller could create a flag that turns a comma-separated string into a slice
// of strings by giving the slice the methods of Value; in particular, Set would
// decompose the comma-separated string into the slice.
func Var(value Value, name string, usage string, required, isset bool) {
	CommandLine.Var(value, name, usage, required, isset)
}

// failf prints to standard error a formatted error and usage message and
// returns the error.
func (f *FlagSet) failf(format string, a ...interface{}) error {
	err := fmt.Errorf(format, a...)
	fmt.Fprintln(f.out(), err)
	f.usage()
	return err
}

// usage calls the Usage method for the flag set if one is specified,
// or the appropriate default usage function otherwise.
func (f *FlagSet) usage() {
	if f.Usage == nil {
		f.defaultUsage()
	} else {
		f.Usage()
	}
}

// parseOne parses one flag. It reports whether a flag was seen.
func (f *FlagSet) parseOne() (bool, error) {
	if len(f.args) == 0 {
		return false, nil
	}
	s := f.args[0]
	if len(s) == 0 || s[0] != '-' || len(s) == 1 {
		return false, nil
	}
	numMinuses := 1
	if s[1] == '-' {
		numMinuses++
		if len(s) == 2 { // "--" terminates the flags
			f.args = f.args[1:]
			return false, nil
		}
	}
	name := s[numMinuses:]
	if len(name) == 0 || name[0] == '-' || name[0] == '=' {
		return false, f.failf("bad flag syntax: %s", s)
	}

	// it's a flag. does it have an argument?
	f.args = f.args[1:]
	hasValue := false
	value := ""
	for i := 1; i < len(name); i++ { // equals cannot be first
		if name[i] == '=' {
			value = name[i+1:]
			hasValue = true
			name = name[0:i]
			break
		}
	}
	m := f.formal
	flag, alreadythere := m[name] // BUG
	if !alreadythere {
		if name == "help" || name == "h" { // special case for nice help message.
			f.usage()
			return false, ErrHelp
		}
		return false, f.failf("flag provided but not defined: -%s", Red(name))
	}

	if fv, ok := flag.Value.(boolFlag); ok && fv.IsBoolFlag() { // special case: doesn't need an arg
		if hasValue {
			if err := fv.Set(value); err != nil {
				return false, f.failf("invalid boolean value %q for -%s: %v", value, name, Red(err))
			}
		} else {
			if err := fv.Set("true"); err != nil {
				return false, f.failf("invalid boolean flag %s: %v", name, Red(err))
			}
		}
	} else {
		// It must have a value, which might be the next argument.
		if !hasValue && len(f.args) > 0 {
			// value is the next arg
			hasValue = true
			value, f.args = f.args[0], f.args[1:]
		}
		if !hasValue {
			return false, f.failf("flag needs an argument: -%s", Red(name))
		}
		if err := flag.Value.Set(value); err != nil {
			return false, f.failf("invalid value %q for flag -%s: %v", value, name, Red(err))
		}
	}
	if f.actual == nil {
		f.actual = make(map[string]*Flag)
	}
	f.actual[name] = flag
	return true, nil
}

// Parse parses flag definitions from the argument list, which should not
// include the command name. Must be called after all flags in the FlagSet
// are defined and before flags are accessed by the program.
// The return value will be ErrHelp if -help or -h were set but not defined.
func (f *FlagSet) Parse(arguments []string) error {
	f.parsed = true
	f.args = arguments
	for {
		seen, err := f.parseOne()
		if seen {
			continue
		}
		if err == nil {
			break
		}
		switch f.errorHandling {
		case ContinueOnError:
			return err
		case ExitOnError:
			os.Exit(2)
		case PanicOnError:
			panic(err)
		}
	}
	return nil
}

// Parsed reports whether f.Parse has been called.
func (f *FlagSet) Parsed() bool {
	return f.parsed
}

// Parse parses the command-line flags from os.Args[1:].  Must be called
// after all flags are defined and before flags are accessed by the program.
func Parse() {
	// Ignore errors; CommandLine is set for ExitOnError.
	CommandLine.Parse(os.Args[1:])
}

// Parsed reports whether the command-line flags have been parsed.
func Parsed() bool {
	return CommandLine.Parsed()
}

// CommandLine is the default set of command-line flags, parsed from os.Args.
// The top-level functions such as BoolVar, Arg, and so on are wrappers for the
// methods of CommandLine.
var CommandLine *FlagSet = NewFlagSet(os.Args[0], ExitOnError)

// SupressFlagUsageMsg suppress flag names in usage when the
// environment variable SUPPRESS_FLAG_USAGE_MSG is set one of the
// following boolean true values: 1,T,t,True,TRUE or true

var SupressFlagUsageMsg = false

// SupressFlagUsageMsgBools is the map of booleans to accept from
// the environment variable SUPPRESS_FLAG_USAGE_MSG
// 1,T,t,True,TRUE or true
var SupressFlagUsageMsgBools = map[string]bool{
	// 1, 0, t, f, T, F, true, false, TRUE, FALSE, True, False
	"1":    true,
	"T":    true,
	"TRUE": true,
	"True": true,
	"t":    true,
	"true": true,
}

func init() {
	// Override generic FlagSet default Usage with call to global Usage.
	// Note: This is not CommandLine.Usage = Usage,
	// because we want any eventual call to use any updated value of Usage,
	// not the value it has when this line is run.
	CommandLine.Usage = commandLineUsage
	var value = os.Getenv("SUPPRESS_FLAG_USAGE_MSG")
	_, SupressFlagUsageMsg = SupressFlagUsageMsgBools[value]
}

func commandLineUsage() {
	Usage()
}

// NewFlagset returns a new, empty flag set with the specified name and
// error handling property.
func NewFlagSet(name string, errorHandling ErrorHandling) *FlagSet {
	f := &FlagSet{
		name:          name,
		errorHandling: errorHandling,
	}
	f.Usage = f.defaultUsage
	return f
}

// Init sets the name and error handling property for a flag set.
// By default, the zero FlagSet uses an empty name and the
// ContinueOnError error handling policy.
func (f *FlagSet) Init(name string, errorHandling ErrorHandling) {
	f.name = name
	f.errorHandling = errorHandling
}

////////////////////////////////////////////////////////////////////////
//
////////////////////////////////////////////////////////////////////////

// -- int8 Value
type int8Value int8

func newInt8Value(val int8, p *int8) *int8Value {
	*p = val
	return (*int8Value)(p)
}

func (i *int8Value) Set(s string) error {
	v, err := strconv.ParseInt(s, 0, 8)
	*i = int8Value(v)
	return err
}

func (i *int8Value) Get() interface{} { return int8(*i) }

func (i *int8Value) String() string { return strconv.FormatInt(int64(*i), 10) }

// Int8Var defines an int8 flag with specified name, default value, and usage string.
// The argument p points to an int8 variable in which to store the value of the flag.
func (f *FlagSet) Int8Var(p *int8, name string, value int8, usage string, required, isset bool) {
	f.Var(newInt8Value(value, p), name, usage, required, isset)
}

// Int8Var defines an int8 flag with specified name, default value, and usage string.
// The argument p points to an int8 variable in which to store the value of the flag.
func Int8Var(p *int8, name string, value int8, usage string, required, isset bool) {
	CommandLine.Var(newInt8Value(value, p), name, usage, required, isset)
}

// Int8 defines an int8 flag with specified name, default value, and usage string.
// The return value is the address of an int8 variable that stores the value of the flag.
func (f *FlagSet) Int8(name string, value int8, usage string, required, isset bool) *int8 {
	p := new(int8)
	f.Int8Var(p, name, value, usage, required, isset)
	return p
}

// Int8 defines an int8 flag with specified name, default value, and usage string.
// The return value is the address of an int8 variable that stores the value of the flag.
func Int8(name string, value int8, usage string, required, isset bool) *int8 {
	return CommandLine.Int8(name, value, usage, required, isset)
}

// -- uint8 Value
type uint8Value uint8

func newUint8Value(val uint8, p *uint8) *uint8Value {
	*p = val
	return (*uint8Value)(p)
}

func (i *uint8Value) Set(s string) error {
	v, err := strconv.ParseUint(s, 0, 8)
	*i = uint8Value(v)
	return err
}

func (i *uint8Value) Get() interface{} { return uint8(*i) }

func (i *uint8Value) String() string { return strconv.FormatUint(uint64(*i), 10) }

// Uint8Var defines a uint8 flag with specified name, default value, and usage string.
// The argument p points to a uint8 variable in which to store the value of the flag.
func (f *FlagSet) Uint8Var(p *uint8, name string, value uint8, usage string, required, isset bool) {
	f.Var(newUint8Value(value, p), name, usage, required, isset)
}

// Uint8Var defines a uint8 flag with specified name, default value, and usage string.
// The argument p points to a uint8 variable in which to store the value of the flag.
func Uint8Var(p *uint8, name string, value uint8, usage string, required, isset bool) {
	CommandLine.Var(newUint8Value(value, p), name, usage, required, isset)
}

// Uint8 defines a uint8 flag with specified name, default value, and usage string.
// The return value is the address of a uint8 variable that stores the value of the flag.
func (f *FlagSet) Uint8(name string, value uint8, usage string, required, isset bool) *uint8 {
	p := new(uint8)
	f.Uint8Var(p, name, value, usage, required, isset)
	return p
}

// Uint8 defines a uint8 flag with specified name, default value, and usage string.
// The return value is the address of a uint8 variable that stores the value of the flag.
func Uint8(name string, value uint8, usage string, required, isset bool) *uint8 {
	return CommandLine.Uint8(name, value, usage, required, isset)
}

////////////////////////////////////////////////////////////////////////
//
////////////////////////////////////////////////////////////////////////

// -- int16 Value
type int16Value int16

func newInt16Value(val int16, p *int16) *int16Value {
	*p = val
	return (*int16Value)(p)
}

func (i *int16Value) Set(s string) error {
	v, err := strconv.ParseInt(s, 0, 16)
	*i = int16Value(v)
	return err
}

func (i *int16Value) Get() interface{} { return int16(*i) }

func (i *int16Value) String() string { return strconv.FormatInt(int64(*i), 10) }

// Int16Var defines an int16 flag with specified name, default value, and usage string.
// The argument p points to an int16 variable in which to store the value of the flag.
func (f *FlagSet) Int16Var(p *int16, name string, value int16, usage string, required, isset bool) {
	f.Var(newInt16Value(value, p), name, usage, required, isset)
}

// Int16Var defines an int16 flag with specified name, default value, and usage string.
// The argument p points to an int16 variable in which to store the value of the flag.
func Int16Var(p *int16, name string, value int16, usage string, required, isset bool) {
	CommandLine.Var(newInt16Value(value, p), name, usage, required, isset)
}

// Int16 defines an int16 flag with specified name, default value, and usage string.
// The return value is the address of an int16 variable that stores the value of the flag.
func (f *FlagSet) Int16(name string, value int16, usage string, required, isset bool) *int16 {
	p := new(int16)
	f.Int16Var(p, name, value, usage, required, isset)
	return p
}

// Int16 defines an int16 flag with specified name, default value, and usage string.
// The return value is the address of an int16 variable that stores the value of the flag.
func Int16(name string, value int16, usage string, required, isset bool) *int16 {
	return CommandLine.Int16(name, value, usage, required, isset)
}

// -- uint16 Value
type uint16Value uint16

func newUint16Value(val uint16, p *uint16) *uint16Value {
	*p = val
	return (*uint16Value)(p)
}

func (i *uint16Value) Set(s string) error {
	v, err := strconv.ParseUint(s, 0, 16)
	*i = uint16Value(v)
	return err
}

func (i *uint16Value) Get() interface{} { return uint16(*i) }

func (i *uint16Value) String() string { return strconv.FormatUint(uint64(*i), 10) }

// Uint16Var defines a uint16 flag with specified name, default value, and usage string.
// The argument p points to a uint16 variable in which to store the value of the flag.
func (f *FlagSet) Uint16Var(p *uint16, name string, value uint16, usage string, required, isset bool) {
	f.Var(newUint16Value(value, p), name, usage, required, isset)
}

// Uint16Var defines a uint16 flag with specified name, default value, and usage string.
// The argument p points to a uint16 variable in which to store the value of the flag.
func Uint16Var(p *uint16, name string, value uint16, usage string, required, isset bool) {
	CommandLine.Var(newUint16Value(value, p), name, usage, required, isset)
}

// Uint16 defines a uint16 flag with specified name, default value, and usage string.
// The return value is the address of a uint16 variable that stores the value of the flag.
func (f *FlagSet) Uint16(name string, value uint16, usage string, required, isset bool) *uint16 {
	p := new(uint16)
	f.Uint16Var(p, name, value, usage, required, isset)
	return p
}

// Uint16 defines a uint16 flag with specified name, default value, and usage string.
// The return value is the address of a uint16 variable that stores the value of the flag.
func Uint16(name string, value uint16, usage string, required, isset bool) *uint16 {
	return CommandLine.Uint16(name, value, usage, required, isset)
}

////////////////////////////////////////////////////////////////////////
//
////////////////////////////////////////////////////////////////////////

// -- int32 Value
type int32Value int32

func newInt32Value(val int32, p *int32) *int32Value {
	*p = val
	return (*int32Value)(p)
}

func (i *int32Value) Set(s string) error {
	v, err := strconv.ParseInt(s, 0, 32)
	*i = int32Value(v)
	return err
}

func (i *int32Value) Get() interface{} { return int32(*i) }

func (i *int32Value) String() string { return strconv.FormatInt(int64(*i), 10) }

// Int32Var defines an int32 flag with specified name, default value, and usage string.
// The argument p points to an int32 variable in which to store the value of the flag.
func (f *FlagSet) Int32Var(p *int32, name string, value int32, usage string, required, isset bool) {
	f.Var(newInt32Value(value, p), name, usage, required, isset)
}

// Int32Var defines an int32 flag with specified name, default value, and usage string.
// The argument p points to an int32 variable in which to store the value of the flag.
func Int32Var(p *int32, name string, value int32, usage string, required, isset bool) {
	CommandLine.Var(newInt32Value(value, p), name, usage, required, isset)
}

// Int32 defines an int32 flag with specified name, default value, and usage string.
// The return value is the address of an int32 variable that stores the value of the flag.
func (f *FlagSet) Int32(name string, value int32, usage string, required, isset bool) *int32 {
	p := new(int32)
	f.Int32Var(p, name, value, usage, required, isset)
	return p
}

// Int32 defines an int32 flag with specified name, default value, and usage string.
// The return value is the address of an int32 variable that stores the value of the flag.
func Int32(name string, value int32, usage string, required, isset bool) *int32 {
	return CommandLine.Int32(name, value, usage, required, isset)
}

// -- uint32 Value
type uint32Value uint32

func newUint32Value(val uint32, p *uint32) *uint32Value {
	*p = val
	return (*uint32Value)(p)
}

func (i *uint32Value) Set(s string) error {
	v, err := strconv.ParseUint(s, 0, 32)
	*i = uint32Value(v)
	return err
}

func (i *uint32Value) Get() interface{} { return uint32(*i) }

func (i *uint32Value) String() string { return strconv.FormatUint(uint64(*i), 10) }

// Uint32Var defines a uint32 flag with specified name, default value, and usage string.
// The argument p points to a uint32 variable in which to store the value of the flag.
func (f *FlagSet) Uint32Var(p *uint32, name string, value uint32, usage string, required, isset bool) {
	f.Var(newUint32Value(value, p), name, usage, required, isset)
}

// Uint32Var defines a uint32 flag with specified name, default value, and usage string.
// The argument p points to a uint32 variable in which to store the value of the flag.
func Uint32Var(p *uint32, name string, value uint32, usage string, required, isset bool) {
	CommandLine.Var(newUint32Value(value, p), name, usage, required, isset)
}

// Uint32 defines a uint32 flag with specified name, default value, and usage string.
// The return value is the address of a uint32 variable that stores the value of the flag.
func (f *FlagSet) Uint32(name string, value uint32, usage string, required, isset bool) *uint32 {
	p := new(uint32)
	f.Uint32Var(p, name, value, usage, required, isset)
	return p
}

// Uint32 defines a uint32 flag with specified name, default value, and usage string.
// The return value is the address of a uint32 variable that stores the value of the flag.
func Uint32(name string, value uint32, usage string, required, isset bool) *uint32 {
	return CommandLine.Uint32(name, value, usage, required, isset)
}
