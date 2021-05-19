package main

import (
	"github.com/kasworld/h4o/appbase"
	"github.com/kasworld/h4o/camera"
	"github.com/kasworld/h4o/gui"
	"github.com/kasworld/h4o/light"
	"github.com/kasworld/h4o/node"
	"github.com/kasworld/h4o/util/framerater"
)

func main() {

}

type GoGLClockCal struct {
	app        *appbase.AppBase
	scene      *node.Node
	cam        *camera.Camera
	camZpos    float32
	pLight     *light.Point
	frameRater *framerater.FrameRater // Render loop frame rater
	labelFPS   *gui.Label             // header FPS label

	sceneAO *node.Node
	sceneCO *node.Node
	sceneDO *node.Node
}
