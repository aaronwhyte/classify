package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"regexp"
	"strings"
)

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func main() {
	pathFlag := flag.String("path", "ERROR", "Path of the file to scan")
	flag.Parse()

	if *pathFlag == "ERROR" {
		panic("Forgot --path <foo.js> ?")
	}

	// Make all paths absolute
	path, err := filepath.Abs(*pathFlag)
	check(err)

	// fmt.Println("reading", path)
	dat, err := ioutil.ReadFile(path)
    check(err)
    js := string(dat);


    // Look for a class and the ctor params
    re := regexp.MustCompile(`function (.*)\((.*)\) {`)
    result := re.FindStringSubmatch(js);
    clazz := ""
    ctorParams := ""
    if len(result) > 0 {
    	clazz = result[1];
		ctorParams = result[2]
    }
    if clazz == "" {
		// fmt.Printf("No class found")
    	fmt.Printf(js)
    	return
    }
	//fmt.Printf("class: %s\n", clazz)
	//fmt.Printf("ctorParams: %s\n", ctorParams)


    // is there a superclass?
    re = regexp.MustCompile(`extends {(.*)}`)
    result = re.FindStringSubmatch(js)
    sup := ""
    if len(result) > 0 {
    	sup = result[1];
    }
	//fmt.Printf("superclass: %s\n", sup)


	if sup == "" {
		// replace "function clazz(ctorParams)" with
		// class clazz {
		//   constructor(ctorParams)
		js = strings.Replace(js,
			"function " + clazz + "(" + ctorParams + ")",
			"class " + clazz + 
				"\n  constructor(" + ctorParams + ")",
			1)
	} else {
		// take care of a bunch of superclass stuff...

		// replace "function clazz(ctorParams)" with
		// class clazz extends sup{
		//   constructor(ctorParams)
		js = strings.Replace(js,
			"function " + clazz + "(" + ctorParams + ")",
			"class " + clazz + " extends " + sup +
				"\n  constructor(" + ctorParams + ")",
			1)
		
		// replace calls to superclass ctor
		// first group is the set of params after "this"
	    re = regexp.MustCompile(sup + `\.call\(this\,?(.*)\);`)
	    result = re.FindStringSubmatch(js)
	    if len(result) > 0 {
	    	args := strings.TrimSpace(result[1])
	    	js = strings.Replace(js,
	    		result[0],
	    		"super(" + args + ");",
	    		1)
	    }

	    // delete prototype pointer and constructor fix:
	    // Foo.prototype = new Bar();
		// Foo.prototype.constructor = Foo;
	    re = regexp.MustCompile(clazz + `\.prototype *= +new ` + sup + `\(.*\)\;\n`)
	    js = re.ReplaceAllString(js, "")
	    re = regexp.MustCompile(clazz + `\.prototype.constructor +=.*;\n`)
	    js = re.ReplaceAllString(js, "")

	    // replace superclass method calls
	    // Bar.prototype.beep.call(this, superblah);
		// ==>
 		// super.beep(superblah);
 		for {
		    re = regexp.MustCompile(sup + `\.prototype\.(.*)\.call\(this\,?(.*)\);`)
		    result = re.FindStringSubmatch(js)
		    if len(result) > 0 {
		    	meth := strings.TrimSpace(result[1])
		    	args := strings.TrimSpace(result[2])
		    	js = strings.Replace(js,
		    		result[0],
		    		"super." + meth + "(" + args + ");",
		    		1)
		    } else {
		    	break
		    }
		}
	}

	// replace "Foo.prototype.bar = function(blah)"
	// with "bar(blah)"
	for {
	    re = regexp.MustCompile(clazz + `\.prototype\.(.*) *= *function.*\((.*)\)`)
	    result = re.FindStringSubmatch(js)
	    if len(result) > 0 {
	    	meth := strings.TrimSpace(result[1])
	    	args := result[2]
	    	js = strings.Replace(js,
	    		result[0],
	    		meth + "(" + args + ")",
	    		1)
	    } else {
	    	break
	    }
	}

	// replace "Foo.SOMETHING"
	// with "static SOMETHING"
	for {
	    re = regexp.MustCompile(clazz + `\.(.*)`)
	    result = re.FindStringSubmatch(js)
	    if len(result) > 0 {
	    	something := result[1]
	    	js = strings.Replace(js,
	    		result[0],
	    		"static " + something,
	    		1)
	    } else {
	    	break
	    }
	}

	// Remove the semicolons. Some of these are wrong to remove, but most are right.
	re = regexp.MustCompile(`\n}\;`)
	js = re.ReplaceAllString(js, "\n}")

	// Remove " * @constructor "
	re = regexp.MustCompile(`\n.*\@constructor.*\n`)
	js = re.ReplaceAllString(js, "\n")

	// Remove " * @extends {...} "
	re = regexp.MustCompile(`\n.*\@extends.*\n`)
	js = re.ReplaceAllString(js, "\n")

	// Remove empty JSDoc
	js = strings.ReplaceAll(js, "/**\n */\n", "")

	// ... and we're done!
	fmt.Printf(js + "\n}\n") // close "class {"
}
