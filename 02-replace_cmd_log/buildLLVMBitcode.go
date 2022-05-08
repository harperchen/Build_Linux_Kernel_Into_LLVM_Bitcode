package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Build one module or whole kernel, e.g., module, kernel
var cmd = "kernel"

// "The path of kernel, e.g., linux"
var path = "/home/weichen/linux-5.15"

// "is -save-temps or not"
// two kinds of two to generate bitcode
var isSaveTemps = false

var CC = filepath.Join(Path, NameClang)
var LD = filepath.Join(Path, NameLD)
var FlagCC = FlagAll + FlagCCNoNumber

const (
	PrefixCmd  = "cmd_"
	SuffixCmd  = ".cmd"
	SuffixCC   = ".o.cmd"
	SuffixLD   = ".a.cmd"
	SuffixLTO  = ".lto.o.cmd"
	NameScript = "build.sh"

	NameClang = "clang"

	// FlagAll -w disable warning
	// FlagAll -g debug info
	FlagAll = " -w -g"

	// FlagCCNoOptzns disable all optimization
	FlagCCNoOptzns = " -mllvm -disable-llvm-optzns"

	// FlagCCNoNumber add label to basic blocks and variables
	FlagCCNoNumber = " -fno-discard-value-names"

	NameLD = "llvm-link"
	FlagLD = " -v"

	// Path of clang and llvm-link
	// Path   = "/home/yhao016/data/benchmark/hang/kernel/toolchain/clang-r353983c/bin/"
	Path = ""

	CmdLinkVmlinux = "llvm-link -v -o built-in.bc arch/x86/kernel/head_64.bc arch/x86/kernel/head64.bc arch/x86/kernel/ebda.bc arch/x86/kernel/platform-quirks.bc init/built-in.bc usr/built-in.bc arch/x86/built-in.bc kernel/built-in.bc certs/built-in.bc mm/built-in.bc fs/built-in.bc ipc/built-in.bc security/built-in.bc crypto/built-in.bc block/built-in.bc lib/built-in.bc arch/x86/lib/built-in.bc lib/lib.bc arch/x86/lib/lib.bc drivers/built-in.bc sound/built-in.bc net/built-in.bc virt/built-in.bc arch/x86/pci/built-in.bc arch/x86/power/built-in.bc arch/x86/video/built-in.bc\n"
	// CmdTools skip the cmd with CmdTools
	CmdTools = "BUILD_STR(s)=$(pound)s"
)

func checkSpecial(res string) string {
	i := strings.Index(res, "BUILD_STR(s)=$(pound)s")
	if i > -1 {
		array := strings.Split(res, " ")
		srcfile := array[len(array)-1]

		if sort.SearchStrings([]string{"bpf.c", "bpf_prog_linfo.c", "btf.c", "btf_dump.c", "gen_loader.c", "hashmap.c", "libbpf.c", "libbpf_errno.c", "xsk.c", "strset.c", "str_error.c", "ringbuf.c", "relo_core.c", "nlattr.c", "netlink.c", "linker.c", "libbpf_probes"}, srcfile) < 17 {
			srcfile = "tools/lib/bpf/" + srcfile
		} else if sort.SearchStrings([]string{"builtin-check.c", "builtin-orc.c", "check.c", "elf.c", "objtool.c", "orc_dump.c", "orc_gen.c", "special.c", "weak.c", "arch/x86/decode.c", "arch/x86/special.c"}, srcfile) < 11 {
			srcfile = "tools/objtool/" + srcfile
		} else if srcfile == "fixdep.c" {
			srcfile = "tools/build/" + srcfile
		} else if sort.SearchStrings([]string{"exec-cmd.c", "help.c", "pager.c", "parse-options.c", "run-command.c", "sigchain.c", "subcmd-config.c"}, srcfile) < 7 {
			srcfile = "tools/lib/subcmd/" + srcfile
		} else if sort.SearchStrings([]string{"../lib/ctype.c", "../lib/rbtree.c", "../lib/string.c", "../lib/std_error_r.c"}, srcfile) < 4 {
			srcfile = strings.Replace(srcfile, "..", "tools", -1)
		} else {

		}
		array[len(array)-1] = srcfile
		res = strings.Join(array, " ")
	}
	return res
}

func getCmd(cmdFilePath string) string {
	res := ""
	if _, err := os.Stat(cmdFilePath); os.IsNotExist(err) {
		fmt.Printf(cmdFilePath + " does not exist\n")
	} else {
		file, err := os.Open(cmdFilePath)
		if err != nil {
			log.Fatal(err)
		}
		defer func(file *os.File) {
			err := file.Close()
			if err != nil {

			}
		}(file)

		scanner := bufio.NewScanner(file)
		scanner.Split(bufio.ScanLines)

		var text []string
		for scanner.Scan() {
			text = append(text, scanner.Text())
		}
		for _, eachLine := range text {
			if strings.HasPrefix(eachLine, PrefixCmd) {
				i := strings.Index(eachLine, ":=")
				// fmt.Println("Index: ", i)
				if i > -1 {
					cmd := eachLine[i+3:]
					res = cmd
				} else {
					fmt.Println("Cmd Index not found")
					fmt.Println(eachLine)
				}
			}
		}
		if err := scanner.Err(); err != nil {
			log.Fatal(err)
		}
	}
	res = checkSpecial(res)
	res += "\n"
	return res
}

func replaceCC(cmd string, addFlag bool) string {
	res := ""
	if addFlag {
		if i := strings.Index(cmd, " -c "); i > -1 {
			if j := strings.Index(cmd, CmdTools); j > -1 {

			} else {
				res += cmd[:i]
				res += FlagCC
				if isSaveTemps {
					res += " -fembed-bitcode -save-temps=obj"
				} else {
					res += " -emit-llvm"
				}
				res += cmd[i:]

				// replace .o to .bc
				if isSaveTemps {

				} else {
					res = strings.Replace(res, ".o ", ".bc ", -1)
				}

				// can not compile .S, so just make a empty bitcode file
				if strings.HasSuffix(cmd, ".S\n") {
					s1 := strings.Split(cmd, " ")
					s2 := s1[len(s1)-1]
					s3 := strings.Split(s2, ".")
					s4 := s3[0]

					res += "\n"
					res = "echo \"\" > " + s4 + ".bc" + "\n"
				}
			}
		} else {
			fmt.Println("CC Index not found")
			fmt.Println(cmd)
		}
	}
	return res
}

func replaceLD(cmd string) string {

	replace := func(cmd string, i int) string {
		res := ""
		cmd = cmd[i+8:]
		if strings.Count(cmd, ".") > 1 {
			res += LD
			res += FlagLD
			res += " -o "
			res += cmd
			if strings.Contains(res, "drivers/of/unittest-data/built-in.o") {
				res = ""
			}
			res = strings.Replace(res, ".o", ".bc", -1)
		} else {
			res = "echo \"\" > " + cmd
		}
		res = strings.Replace(res, ".a ", ".bc ", -1)
		res = strings.Replace(res, ".a\n", ".bc\n", -1)
		// for this drivers/misc/lkdtm/rodata.bc
		res = strings.Replace(res, "rodata_objcopy.bc", "rodata.bc", -1)
		res = strings.Replace(res, " drivers/of/unittest-data/built-in.bc", "", -1)
		return res
	}

	res := ""
	// fmt.Println("Index: ", i)
	if i := strings.Index(cmd, " rcSTPD"); i > -1 {
		res = replace(cmd, i)
	} else if i := strings.Index(cmd, " cDPrST"); i > -1 {
		res = replace(cmd, i)
	} else if i := strings.Index(cmd, " cDPrsT"); i > -1 {
		res = replace(cmd, i)
	} else {
		fmt.Println("LD Index not found")
		fmt.Println(cmd)
	}

	return res
}

func get_linked_target(cmd string) string {
	res := ""
	if strings.Contains(cmd, "llvm-link -v -o") {
		res = cmd[len("llvm-link -v -o") : strings.Index(cmd, ".bc")+3]
	}
	return res
}

func buildModule(moduleDirPath string) string {
	res1 := ""
	err := filepath.Walk(moduleDirPath,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if strings.HasSuffix(info.Name(), SuffixCC) {
				cmd := getCmd(path)
				if strings.HasPrefix(cmd, NameClang) {
					res2 := replaceCC(cmd, true)
					res2 = strings.Replace(res2, NameClang, CC, -1)
					//res2 = strings.Replace(res2, IncludeOld, IncludeNew, -1)
					res1 += res2
				} else {
					// fmt.Println("clang not found")
					// fmt.Println(path)
					// fmt.Println(cmd)
				}
			}
			return nil
		})
	if err != nil {
		log.Println(err)
	}

	res2 := ""
	module_file := ""
	err = filepath.Walk(moduleDirPath,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if strings.HasSuffix(info.Name(), SuffixLD) {
				cmd := getCmd(path)
				res2 = replaceLD(cmd) + res2
			}
			// for kernel module (*.ko, *.lto)
			if strings.HasSuffix(info.Name(), SuffixCC) {
				cmd := getCmd(path)
				res2 = replaceLD(cmd) + res2
			}
			if strings.HasSuffix(info.Name(), SuffixLTO) {
				cmd := getCmd(path)
				cmd = cmd[strings.Index(cmd, "--whole-archive")+len("--whole-archive") : len(cmd)-1]
				cmd = strings.Replace(cmd, ".o", ".bc", -1)
				module_file = cmd + module_file
			}
			return nil
		})
	if err != nil {
		log.Println(err)
	}
	fmt.Println("module_file")
	fmt.Println(module_file)

	return res1 + res2
}

func generateScript(path string, cmd string) {
	res := "#!/bin/bash\n"
	res += cmd

	pathScript := filepath.Join(path, NameScript)
	_ = os.Remove(pathScript)
	fmt.Printf("script path : bash %s\n", pathScript)
	f, err := os.OpenFile(pathScript, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		log.Println(err)
	}
	defer func(f *os.File) {
		err := f.Close()
		if err != nil {

		}
	}(f)

	_, _ = f.WriteString(res)
}

func main() {
	switch cmd {
	case "module":
		{
			fmt.Printf("Build one module\n")
			res := buildModule(path)
			generateScript(path, res)
		}
	case "kernel":
		{
			fmt.Printf("Build whole kernel\n")
			res := buildModule(path)
			res += CmdLinkVmlinux
			generateScript(path, res)
		}
	default:
		fmt.Printf("cmd is invalid\n")
	}
}
