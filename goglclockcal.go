package main

import (
	"context"
	"fmt"
	"time"

	"github.com/kasworld/h4o/appbase"
	"github.com/kasworld/h4o/appbase/appwindow"
	"github.com/kasworld/h4o/camera"
	"github.com/kasworld/h4o/eventtype"
	"github.com/kasworld/h4o/gls"
	"github.com/kasworld/h4o/gui"
	"github.com/kasworld/h4o/light"
	"github.com/kasworld/h4o/math32"
	"github.com/kasworld/h4o/node"
	"github.com/kasworld/h4o/renderer"
	"github.com/kasworld/h4o/util/framerater"
	"github.com/kasworld/h4o/util/helper"
)

func main() {
	NewMtLogic().Run()
}

const (
	BufferSize = 10
)

// multi thead logic
type MtLogic struct {
	doClose func() `prettystring:"hide"`

	// logic to view channel
	l2vCh chan interface{}
	// view to logic channel
	v2lCh chan interface{}
}

func NewMtLogic() *MtLogic {
	rtn := &MtLogic{
		l2vCh: make(chan interface{}, BufferSize),
		v2lCh: make(chan interface{}, BufferSize),
	}
	return rtn
}

func (ls *MtLogic) handleV2LCh() {
	for fromView := range ls.v2lCh {
		switch pk := fromView.(type) {
		default:
			fmt.Printf("unknown packet %v", pk)
			ls.doClose()

			// handle known packet from view
		}
	}
}

func (ls *MtLogic) Run() {
	ctx, closeCtx := context.WithCancel(context.Background())
	ls.doClose = closeCtx
	defer closeCtx()

	go func() {
		// now run single thread view
		err := NewStView(ls.l2vCh, ls.v2lCh).Run()
		if err != nil {
			fmt.Printf("view err %v", err)
		}
		ls.doClose()
	}()

	go ls.handleV2LCh()
	timerInfoTk := time.NewTicker(1 * time.Second)
	defer timerInfoTk.Stop()
loop:
	for {
		select {
		case <-ctx.Done():
			break loop
		case <-timerInfoTk.C:
			if len(ls.v2lCh) >= cap(ls.v2lCh) {
				fmt.Printf("v2lCh full %v/%v", len(ls.v2lCh), cap(ls.v2lCh))
				break loop
			}
			if len(ls.l2vCh) >= cap(ls.l2vCh) {
				fmt.Printf("l2vCh full %v/%v", len(ls.l2vCh), cap(ls.l2vCh))
				break loop
			}
		}
	}
}

// single thread view
// runtime.LockOSThread
type StView struct {
	// logic to view channel
	l2vCh chan interface{}
	// view to logic channel
	v2lCh chan interface{}

	appBase    *appbase.AppBase
	scene      *node.Node
	cam        *camera.Camera
	camZpos    float32
	pLight     *light.Point
	frameRater *framerater.FrameRater // Render loop frame rater
	labelFPS   *gui.Label             // header FPS label

}

func NewStView(l2vCh chan interface{}, v2lCh chan interface{}) *StView {
	rtn := &StView{
		l2vCh: l2vCh,
		v2lCh: v2lCh,
	}
	return rtn
}

func (cv *StView) Run() error {
	if err := cv.glInit(); err != nil {
		return err
	}
	// Set background color to gray
	cv.appBase.Gls().ClearColor(0.5, 0.5, 0.5, 1.0)
	cv.appBase.Run(cv.updateGL)

	return nil
}
func (cv *StView) updateGL(renderer *renderer.Renderer, deltaTime time.Duration) {
	// Start measuring this frame
	cv.frameRater.Start()

	cv.appBase.Gls().Clear(gls.DEPTH_BUFFER_BIT | gls.STENCIL_BUFFER_BIT | gls.COLOR_BUFFER_BIT)
	renderer.Render(cv.scene, cv.cam)
	cv.handle_l2vCh()

	// Control and update FPS
	cv.frameRater.Wait()
	cv.updateFPS()

}

func (cv *StView) glInit() error {
	// Create application and scene
	cv.appBase = appbase.New("h4o clock calendar", 1920, 1080)
	cv.scene = node.NewNode()

	// Set the scene to be managed by the gui manager
	gui.Manager().Set(cv.scene)

	// Create perspective camera
	cv.cam = camera.New(1)
	cv.cam.SetFar(1400)
	cv.camZpos = 100
	cv.cam.SetPosition(0, 0, cv.camZpos)
	cv.scene.Add(cv.cam)

	// Set up orbit control for the camera
	// camera.NewOrbitControl(cv.cam)

	cv.appBase.Subscribe(eventtype.OnWindowSize, cv.onResize)
	cv.onResize(eventtype.OnResize, nil)

	// Create and add lights to the scene
	cv.scene.Add(light.NewAmbient(&math32.Color{1.0, 1.0, 1.0}, 0.8))
	cv.pLight = light.NewPoint(&math32.Color{1, 1, 1}, 5.0)
	cv.pLight.SetPosition(1, 0, 2)
	cv.scene.Add(cv.pLight)

	// Create and add an axis helper to the scene
	cv.scene.Add(helper.NewAxes(100))

	cv.frameRater = framerater.NewFrameRater(60)
	cv.labelFPS = gui.NewLabel(" ")
	cv.labelFPS.SetFontSize(20)
	cv.labelFPS.SetLayoutParams(&gui.HBoxLayoutParams{AlignV: gui.AlignCenter})
	lightTextColor := math32.Color4{0.8, 0.8, 0.8, 1}
	cv.labelFPS.SetColor4(&lightTextColor)
	cv.scene.Add(cv.labelFPS)

	gui.Manager().SubscribeID(eventtype.OnMouseUp, cv, cv.onMouse)
	gui.Manager().SubscribeID(eventtype.OnMouseDown, cv, cv.onMouse)
	gui.Manager().SubscribeID(eventtype.OnScroll, &cv, cv.onScroll)

	return nil
}

func (cv *StView) handle_l2vCh() {
	for len(cv.l2vCh) > 0 {
		fromLogic := <-cv.l2vCh
		switch pk := fromLogic.(type) {
		default:
			fmt.Printf("unknown packet %v", pk)

			// handle known packet from logic
		}
	}
}

// Set up callback to update viewport and camera aspect ratio when the window is resized
func (cv *StView) onResize(evname eventtype.EventType, ev interface{}) {
	// Get framebuffer size and update viewport accordingly
	width, height := cv.appBase.GetSize()
	cv.appBase.Gls().Viewport(0, 0, int32(width), int32(height))
	// Update the camera's aspect ratio
	cv.cam.SetAspect(float32(width) / float32(height))
}

// UpdateFPS updates the fps value in the window title or header label
func (cv *StView) updateFPS() {

	// Get the FPS and potential FPS from the frameRater
	fps, pfps, ok := cv.frameRater.FPS(time.Duration(60) * time.Millisecond)
	if !ok {
		return
	}

	// Show the FPS in the header label
	cv.labelFPS.SetText(fmt.Sprintf("%3.1f / %3.1f", fps, pfps))
}

// onMouse is called when an OnMouseDown/OnMouseUp event is received.
func (cv *StView) onMouse(evname eventtype.EventType, ev interface{}) {

	switch evname {
	case eventtype.OnMouseDown:
		// gui.Manager().SetCursorFocus(cv)
		mev := ev.(*appwindow.MouseEvent)
		switch mev.Button {
		case appwindow.MouseButtonLeft: // Rotate
		case appwindow.MouseButtonMiddle: // Zoom
		case appwindow.MouseButtonRight: // Pan
		}
	case eventtype.OnMouseUp:
		// gui.Manager().SetCursorFocus(nil)
	}
}

// onScroll is called when an OnScroll event is received.
func (cv *StView) onScroll(evname eventtype.EventType, ev interface{}) {
	zF := float32(1.5)
	sev := ev.(*appwindow.ScrollEvent)
	if sev.Yoffset > 0 {
		cv.camZpos *= zF
		if cv.camZpos > 1000 {
			cv.camZpos = 1000
		}
	} else if sev.Yoffset < 0 {
		cv.camZpos /= zF
		if cv.camZpos < 10 {
			cv.camZpos = 10
		}
	}
}
