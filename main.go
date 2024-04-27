package main

import (
	"errors"
	"fmt"
	"golang.org/x/sys/windows/registry"
	"io/fs"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
)

const (
	Registration16XCS = "Software\\PremiumSoft\\NavicatPremium\\Registration16XCS"
	Update            = "Software\\PremiumSoft\\NavicatPremium\\Update"
	CLSID             = "Software\\Classes\\CLSID"
)

func main() {
	var (
		err     error
		key     registry.Key
		subKeys []string
	)

	err = registry.DeleteKey(registry.CURRENT_USER, Registration16XCS)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		panic(fmt.Sprintf("%s->%v", Registration16XCS, err))
		return
	}

	err = registry.DeleteKey(registry.CURRENT_USER, Update)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		panic(fmt.Sprintf("%s->%v", Update, err))
		return
	}

	key, err = registry.OpenKey(registry.CURRENT_USER, CLSID, registry.ALL_ACCESS)
	if err != nil {
		panic(fmt.Sprintf("打开key异常[%s]->%v", CLSID, err))
		return
	}

	subKeys, err = key.ReadSubKeyNames(-1)
	if err != nil {
		panic(fmt.Sprintf("获取所有子key异常[%s]->%v", CLSID, err))
		return
	}

	defer key.Close()

	err = registry.DeleteKey(registry.CURRENT_USER, "Software\\Classes\\CLSID\\{8F840E3C-3150-CA28-46A0-0C0465E7D497}")
	if err != nil {
		println(err.Error())
	}

	var wg sync.WaitGroup
	for _, subKey := range subKeys {
		wg.Add(1)
		go func(subKey string) {
			defer wg.Done()
			var (
				allSubKeys   []string
				subKeyHandle registry.Key
				realPath     = fmt.Sprintf("%s\\%s", CLSID, subKey)
			)
			subKeyHandle, err = registry.OpenKey(registry.CURRENT_USER, realPath, registry.ALL_ACCESS)

			if err != nil {
				println(fmt.Sprintf("打开key异常[%s]->%v", realPath, err))
				return
			}

			allSubKeys, err = subKeyHandle.ReadSubKeyNames(-1)
			if err != nil {
				println(fmt.Sprintf("获取所有子key异常[%s]->%v", realPath, err))
				return
			}

			var needDel bool
			for _, item := range allSubKeys {
				if strings.Index(item, "Info") != -1 || strings.Index(item, "ShellFolder") != -1 {
					needDel = true
					subKeyHandle.Close()
					break
				}
			}

			if needDel {
				for _, item := range allSubKeys {
					println(fmt.Sprintf("%s\\%s", realPath, item))
					registry.DeleteKey(registry.CURRENT_USER, fmt.Sprintf("%s\\%s", realPath, item))
				}

				err = registry.DeleteKey(registry.CURRENT_USER, realPath)
				if err != nil {
					println(err.Error())
				}
			}

		}(subKey)
	}
	wg.Wait()

	fmt.Println("Press Ctrl+C again to exit.")
	// 创建一个通道来接收操作系统的信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT)
	select {
	case <-sigCh:
		// 第二次信号，退出程序
		os.Exit(0)
	}
}
