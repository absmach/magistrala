// +build alltests
// +build go1.7

package codec

// Run this using:
//   go test -tags=alltests -run=Suite -coverprofile=cov.out
//   go tool cover -html=cov.out
//
// Because build tags are a build time parameter, we will have to test out the
// different tags separately.
// Tags: x codecgen safe appengine notfastpath
//
// These tags should be added to alltests, e.g.
//   go test '-tags=alltests x codecgen' -run=Suite -coverprofile=cov.out
//
// To run all tests before submitting code, run:
//    a=( "" "safe" "codecgen" "notfastpath" "codecgen notfastpath" "codecgen safe" "safe notfastpath" )
//    for i in "${a[@]}"; do echo ">>>> TAGS: $i"; go test "-tags=alltests $i" -run=Suite; done
//
// This only works on go1.7 and above. This is when subtests and suites were supported.

import "testing"

// func TestMain(m *testing.M) {
// 	println("calling TestMain")
// 	// set some parameters
// 	exitcode := m.Run()
// 	os.Exit(exitcode)
// }

func testSuite(t *testing.T, f func(t *testing.T)) {
	// find . -name "*_test.go" | xargs grep -e 'flag.' | cut -d '&' -f 2 | cut -d ',' -f 1 | grep -e '^test'
	// Disregard the following: testVerbose, testInitDebug, testSkipIntf, testJsonIndent (Need a test for it)

	testReinit() // so flag.Parse() is called first, and never called again

	testDecodeOptions = DecodeOptions{}
	testEncodeOptions = EncodeOptions{}

	testUseMust = false
	testCanonical = false
	testUseMust = false
	testInternStr = false
	testUseIoEncDec = false
	testStructToArray = false
	testWriteNoSymbols = false
	testCheckCircRef = false
	testJsonHTMLCharsAsIs = false
	testUseReset = false
	testMaxInitLen = 0
	testJsonIndent = 0
	testUseIoWrapper = false
	testNumRepeatString = 8

	testReinit()
	t.Run("optionsFalse", f)

	testMaxInitLen = 10
	testJsonIndent = 8
	testReinit()
	t.Run("initLen10-jsonSpaces", f)

	testReinit()
	testMaxInitLen = 10
	testJsonIndent = -1
	testReinit()
	t.Run("initLen10-jsonTabs", f)

	testCanonical = true
	testUseMust = true
	testInternStr = true
	testUseIoEncDec = true
	testStructToArray = true
	testWriteNoSymbols = true
	testCheckCircRef = true
	testJsonHTMLCharsAsIs = true
	testUseReset = true
	testDecodeOptions.MapValueReset = true
	testReinit()
	t.Run("optionsTrue", f)

	testUseIoWrapper = true
	testReinit()
	t.Run("optionsTrue-ioWrapper", f)

	testDepth = 6
	testReinit()
	t.Run("optionsTrue-deepstruct", f)

	// make buffer small enough so that we have to re-fill multiple times.
	testSkipRPCTests = true
	testUseIoEncDec = true
	testDecodeOptions.ReaderBufferSize = 128
	testEncodeOptions.WriterBufferSize = 128
	testReinit()
	t.Run("optionsTrue-bufio", f)
	testDecodeOptions.ReaderBufferSize = 0
	testEncodeOptions.WriterBufferSize = 0
	testSkipRPCTests = false

	testNumRepeatString = 32
	testReinit()
	t.Run("optionsTrue-largestrings", f)

	// The following here MUST be tested individually, as they create
	// side effects i.e. the decoded value is different.
	// testDecodeOptions.MapValueReset = true // ok - no side effects
	// testDecodeOptions.InterfaceReset = true // error??? because we do deepEquals to verify
	// testDecodeOptions.ErrorIfNoField = true // error, as expected, as fields not there
	// testDecodeOptions.ErrorIfNoArrayExpand = true // no error, but no error case either
	// testDecodeOptions.PreferArrayOverSlice = true // error??? because slice != array.
	// .... however, update deepEqual to take this option
	// testReinit()
	// t.Run("optionsTrue-resetOptions", f)
}

/*
z='codec_test.go'
find . -name "$z" | xargs grep -e '^func Test' | \
    cut -d '(' -f 1 | cut -d ' ' -f 2 | \
    while read f; do echo "t.Run(\"$f\", $f)"; done
*/

func testCodecGroup(t *testing.T) {
	// println("running testcodecsuite")
	// <setup code>

	t.Run("TestBincCodecsTable", TestBincCodecsTable)
	t.Run("TestBincCodecsMisc", TestBincCodecsMisc)
	t.Run("TestBincCodecsEmbeddedPointer", TestBincCodecsEmbeddedPointer)
	t.Run("TestBincStdEncIntf", TestBincStdEncIntf)
	t.Run("TestBincMammoth", TestBincMammoth)
	t.Run("TestSimpleCodecsTable", TestSimpleCodecsTable)
	t.Run("TestSimpleCodecsMisc", TestSimpleCodecsMisc)
	t.Run("TestSimpleCodecsEmbeddedPointer", TestSimpleCodecsEmbeddedPointer)
	t.Run("TestSimpleStdEncIntf", TestSimpleStdEncIntf)
	t.Run("TestSimpleMammoth", TestSimpleMammoth)
	t.Run("TestMsgpackCodecsTable", TestMsgpackCodecsTable)
	t.Run("TestMsgpackCodecsMisc", TestMsgpackCodecsMisc)
	t.Run("TestMsgpackCodecsEmbeddedPointer", TestMsgpackCodecsEmbeddedPointer)
	t.Run("TestMsgpackStdEncIntf", TestMsgpackStdEncIntf)
	t.Run("TestMsgpackMammoth", TestMsgpackMammoth)
	t.Run("TestCborCodecsTable", TestCborCodecsTable)
	t.Run("TestCborCodecsMisc", TestCborCodecsMisc)
	t.Run("TestCborCodecsEmbeddedPointer", TestCborCodecsEmbeddedPointer)
	t.Run("TestCborMapEncodeForCanonical", TestCborMapEncodeForCanonical)
	t.Run("TestCborCodecChan", TestCborCodecChan)
	t.Run("TestCborStdEncIntf", TestCborStdEncIntf)
	t.Run("TestCborMammoth", TestCborMammoth)
	t.Run("TestJsonCodecsTable", TestJsonCodecsTable)
	t.Run("TestJsonCodecsMisc", TestJsonCodecsMisc)
	t.Run("TestJsonCodecsEmbeddedPointer", TestJsonCodecsEmbeddedPointer)
	t.Run("TestJsonCodecChan", TestJsonCodecChan)
	t.Run("TestJsonStdEncIntf", TestJsonStdEncIntf)
	t.Run("TestJsonMammoth", TestJsonMammoth)
	t.Run("TestJsonRaw", TestJsonRaw)
	t.Run("TestBincRaw", TestBincRaw)
	t.Run("TestMsgpackRaw", TestMsgpackRaw)
	t.Run("TestSimpleRaw", TestSimpleRaw)
	t.Run("TestCborRaw", TestCborRaw)
	t.Run("TestAllEncCircularRef", TestAllEncCircularRef)
	t.Run("TestAllAnonCycle", TestAllAnonCycle)
	t.Run("TestBincRpcGo", TestBincRpcGo)
	t.Run("TestSimpleRpcGo", TestSimpleRpcGo)
	t.Run("TestMsgpackRpcGo", TestMsgpackRpcGo)
	t.Run("TestCborRpcGo", TestCborRpcGo)
	t.Run("TestJsonRpcGo", TestJsonRpcGo)
	t.Run("TestMsgpackRpcSpec", TestMsgpackRpcSpec)
	t.Run("TestBincUnderlyingType", TestBincUnderlyingType)
	t.Run("TestJsonLargeInteger", TestJsonLargeInteger)
	t.Run("TestJsonDecodeNonStringScalarInStringContext", TestJsonDecodeNonStringScalarInStringContext)
	t.Run("TestJsonEncodeIndent", TestJsonEncodeIndent)
	// <tear-down code>
}

func TestCodecSuite(t *testing.T) { testSuite(t, testCodecGroup) }

// func TestCodecSuite(t *testing.T) { testSuite2(t, testCodecGroup2) }
// func testCodecGroup2(t *testing.T) {
// 	t.Run("TestJsonCodecsTable", TestJsonCodecsTable)
// 	t.Run("TestJsonCodecsMisc", TestJsonCodecsMisc)
// }
// func testSuite2(t *testing.T, f func(t *testing.T)) {
// 	testUseIoEncDec = true
// 	testDecodeOptions = DecodeOptions{}
// 	testEncodeOptions = EncodeOptions{}
// 	testDecodeOptions.ReaderBufferSize = 128
// 	testEncodeOptions.WriterBufferSize = 128
// 	testReinit()
// 	t.Run("optionsTrue-bufio", f)
// }
