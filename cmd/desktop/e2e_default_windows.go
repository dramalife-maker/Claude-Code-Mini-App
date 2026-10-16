//go:build windows && !e2e

package main

import "github.com/wailsapp/wails/v3/pkg/application"

// e2eApplyOptions 在一般與正式建置中不做任何修改。
func e2eApplyOptions(*application.Options) {}

// e2eShutdownHang 在一般與正式建置中不做任何事。
func e2eShutdownHang() {}
