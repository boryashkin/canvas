'use client';

import { PointerEvent, useEffect, useState } from "react";


let ws: WebSocket;

let coordsX = new Map<number, number[]>();
let coordsY = new Map<number, number[]>();
let receivedCoordsCounter = 0
let renderedCoordsCounter = 0
let lastSentX: number;
let lastSentY: number;
let currentZoom = 1;
let leftOffset = 0;
let topOffset = 0;

const offsetStep = 100;
const maxWidth = 8192;
const maxHeight = 8192;
const renderFreq = 50;

const colors = [
    "#000000", "#eded01", "#1CE6FF", "#FF34FF", "#FF4A46", "#008941", "#006FA6", "#A30059",
    "#FFDBE5", "#7A4900", "#0000A6", "#63FFAC", "#B79762", "#004D43", "#8FB0FF", "#997D87",
    "#5A0007", "#809693", "#dee09d", "#1B4400", "#4FC601", "#3B5DFF", "#4A3B53", "#FF2F80",
    "#61615A", "#BA0900", "#6B7900", "#00C2A0", "#FFAA92", "#FF90C9", "#B903AA", "#D16100",
    "#DDEFFF", "#000035", "#7B4F4B", "#A1C299", "#300018", "#0AA6D8", "#013349", "#00846F",
    "#372101", "#FFB500", "#C2FFED", "#A079BF", "#CC0744", "#C0B9B2", "#C2FF99", "#001E09",
    "#00489C", "#6F0062", "#0CBD66", "#EEC3FF", "#456D75", "#B77B68", "#7A87A1", "#788D66",
    "#885578", "#FAD09F", "#FF8A9A", "#D157A0", "#BEC459", "#456648", "#0086ED", "#886F4C",

    "#34362D", "#B4A8BD", "#00A6AA", "#452C2C", "#636375", "#A3C8C9", "#FF913F", "#938A81",
    "#575329", "#00FECF", "#B05B6F", "#8CD0FF", "#3B9700", "#04F757", "#C8A1A1", "#1E6E00",
    "#7900D7", "#A77500", "#6367A9", "#A05837", "#6B002C", "#772600", "#D790FF", "#9B9700",
    "#549E79", "#FFF69F", "#201625", "#72418F", "#BC23FF", "#99ADC0", "#3A2465", "#922329",
    "#5f5752", "#FDE8DC", "#404E55", "#0089A3", "#CB7E98", "#A4E804", "#324E72", "#6A3A4C",
    "#83AB58", "#001C1E", "#D1F7CE", "#004B28", "#C8D0F6", "#A3A489", "#806C66", "#222800",
    "#BF5650", "#E83000", "#66796D", "#DA007C", "#FF1A59", "#8ADBB4", "#1E0200", "#5B4E51",
    "#C895C5", "#320033", "#FF6832", "#66E1D3", "#CFCDAC", "#D0AC94", "#7ED379", "#012C58"
];

const checkEncoded = (x: number) => {
    return x > 49152 && (x ^ prevEncoding) < maxWidth
}

const adjustViewpointWidth = (x: number): number => {
    x -= leftOffset
    if (x * currentZoom > document.body.clientWidth) {
        return -1
    }
    return x * currentZoom
}
const adjustViewpointHeight = (y: number): number => {
    y -= topOffset
    if (y * currentZoom > document.body.clientHeight) {
        return -1
    }
    return y * currentZoom
}
const convertToAbsoluteWidth = (x: number): number => {
    return roundNum(x / currentZoom) + leftOffset
}
const convertToAbsoluteHeight = (y: number): number => {
    return roundNum(y / currentZoom) + topOffset
}

const zoomIn = () => {
    let zoom = currentZoom
    if (zoom > 0 && zoom < 1) {
        zoom = roundNum(zoom + 0.1)
        setCurrentZoom(zoom)
        return
    }
    if (zoom > 10) {
        zoom = Math.round(zoom + 2)
        setCurrentZoom(zoom)
        return
    }
    zoom = roundNum(zoom + 0.5)
    setCurrentZoom(zoom)
}

const setCurrentZoom = (zoom: number) => {
    if (zoom < 0.1) {
        currentZoom = 0.1
        return
    }
    if (zoom > 100) {
        currentZoom = 100
    }
    currentZoom = zoom
}

const zoomOut = () => {
    let zoom = currentZoom
    if (zoom > 0.2 && zoom < 1) {
        zoom = roundNum(zoom - 0.1)
        setCurrentZoom(zoom)
        return
    }
    if (zoom > 10) {
        zoom = Math.round(zoom - 2)
        setCurrentZoom(zoom)
        return
    }
    zoom = roundNum(zoom - 0.5)
    setCurrentZoom(zoom)
}

const roundNum = (x: number): number => {
    return Math.round(x * 10) / 10
}

// bits to point that a coordinates pair is connected to a previous
const prevEncoding = 0b1100000000000000

export const Game = (props: {gameId: string, websocketProto: string, websocketPort: string}) => {
    const [wsState, setWsState] = useState(0)
    const [zoom, setZoom] = useState(1)
    console.log("rerender")
    useEffect(() => {
        let canvas = document.getElementById("my-canvas") as HTMLCanvasElement;
        let context = canvas.getContext("2d");
        if (!context) {
            return
        }
        
        canvas.width = document.body.clientWidth;
        canvas.height = document.body.clientHeight;
        currentZoom = zoom
        console.log("current zoom", currentZoom)

        context.lineCap = 'round';
        context.lineJoin = 'round';
        context.strokeStyle = 'black';
        context.lineWidth = 1;

        const renderFunction = () => {
            if (!context) {
                return
            }
            if (receivedCoordsCounter == renderedCoordsCounter) {
                return
            }
            context.clearRect(0, 0, canvas.width, canvas.height)
            renderedCoordsCounter = receivedCoordsCounter
            coordsX.forEach((value, key) => {
                if (!context) {
                    console.log("no context")
                    return
                }
                let valueY = coordsY.get(key)
                if (!valueY) {
                    console.log("no valueY")
                    return
                }
                
                context.strokeStyle = colors[key % colors.length]
                for (let ii = 0; ii < value.length; ii++) {
                    context.beginPath();
                    let rawX = value[ii]
                    let rawY = valueY[ii]
                    let x = rawX
                    let x0 = rawX;
                    let y0 = rawY;
                    
                    
                    let prev = false
                    if (checkEncoded(rawX)) {
                        x ^= prevEncoding
                        prev = ii > 0
                    }

                    if (prev) {
                        let prevX = value[ii - 1]
                        if (checkEncoded(prevX)) {
                            prevX ^= prevEncoding
                        }
                        prevX = adjustViewpointWidth(prevX)
                        let prevY = adjustViewpointHeight(valueY[ii - 1])
                        if (prevX < 0 || prevY < 0) {
                            continue
                        }
                        x0 = prevX
                        y0 = prevY
                    } else {
                        x0 = adjustViewpointWidth(x)
                        y0 = adjustViewpointHeight(rawY)
                        if (x0 < 0 || y0 < 0) {
                            continue
                        }
                    }
                    x = adjustViewpointWidth(x)
                    let y = adjustViewpointHeight(valueY[ii])
                    if (x < 0 || y < 0) {
                        continue
                    }
                    context.moveTo(x0, y0);
                    context.lineTo(x, y);
                    context.stroke()

                }
                context.closePath();
            })


            if (i % 1000 == 0) {
                console.log("rendered", i)
            }
            i++
        }
        let i = 0
        let renderLoopId = setInterval(renderFunction, renderFreq)


        let wsHost = props.websocketProto + "://" + window.location.host.split(":")[0] + ":" + props.websocketPort
        ws = new WebSocket(wsHost + "/ws/canvas/" + props.gameId)
        ws.addEventListener("open", (event) => {
            ws.send("Hello Server!");
            setWsState(1)
        });
        ws.addEventListener("close", (event) => {
            console.log("failed to connect")
            setWsState(-1)
            // no need to run that often
            clearInterval(renderLoopId)
            // but still run it to be able to use zoom
            setInterval(renderFunction, renderFreq * 5)
        });

        // Listen for messages
        ws.addEventListener("message", (event) => {
            if (!context) {
                return
            }
            let data = event.data as Blob

            if (typeof data == "string") {
                console.log("data", data)
                return
            }
            data.arrayBuffer().then((b) => {
                receivedCoordsCounter++
                let ints = new Uint16Array(b)
                if (!coordsX.has(ints[0])) {
                    coordsX.set(ints[0], [])
                    coordsY.set(ints[0], [])
                }
                coordsX.get(ints[0])?.push(ints[1])
                coordsY.get(ints[0])?.push(ints[2])
            }).catch(() => {
                console.log("error while resolving")
            })
        });
    }, [])

    const changeZoom = (doZoomIn: boolean) => {
        if (doZoomIn) {
            setZoom(() => {
                zoomIn()
                return currentZoom
            })
            renderedCoordsCounter--;
            return
        }
        setZoom(() => {
            zoomOut()
            return currentZoom
        }) 
        renderedCoordsCounter--;
    }

    const saveMove = (e: PointerEvent) => {
        let isPressedCrossbrowser = e.pressure || e.buttons
        if (!isPressedCrossbrowser) {
            return
        }

        if (wsState != 1) { return }
        e.preventDefault();
        const buffer = new ArrayBuffer(4);
        const view = new Uint16Array(buffer);
        if (e.clientX > maxWidth || e.clientY > maxHeight) {
            return
        }
        view[0] = convertToAbsoluteWidth(e.clientX)
        view[1] = convertToAbsoluteHeight(e.clientY)
        if (view[0] == lastSentX && lastSentY == view[1]) {
            return
        }
        lastSentX = view[0]
        lastSentY = view[1]
        view[0] |= prevEncoding // it was wrapped by this condition e.movementX != 0 || e.movementY != 0
        // but no longer needed since onPressedUp and Down are sending no-prevEncoding to separate encoded (connected) lines

        ws.send(view)
    }


    return (<div className="bg-red-50 w-screen h-screen overflow-hidden fixed">
        <div style={{display: wsState == 1 ? "none" : "block"}} className="fixed top-0 left-0 bg-blue-300 text-white z-50">
            {wsState != -1 ? "Start drawing..." : "Connection error"}<br/><i>{wsState != -1 ? "After loading" : "Refresh the page"}</i></div>
        <div className="fixed bottom-0 left-0 flex z-50">
            <button className="p-2 mr-2 bg-blue-300 text-white relative " onClick={() => {changeZoom(true); console.log(currentZoom)}}>+</button>
            <ZoomBlock zoom={zoom} />
            <button className="p-2 mr-2 bg-blue-300 text-white relative" onClick={() => {changeZoom(false); console.log(currentZoom)}}>-</button>
            <button className="p-2 mr-2 bg-blue-300 text-white relative" onClick={() => {leftOffset+=offsetStep; renderedCoordsCounter--; console.log("left", leftOffset)}}>➡️</button>
            <button className="p-2 mr-2 bg-blue-300 text-white relative" onClick={() => {leftOffset-=offsetStep; renderedCoordsCounter--; console.log("left", leftOffset)}}>⬅️</button>
            <button className="p-2 mr-2 bg-blue-300 text-white relative" onClick={() => {topOffset+=offsetStep; renderedCoordsCounter--; console.log("top", topOffset)}}>⬇️</button>
            <button className="p-2 mr-2 bg-blue-300 text-white relative" onClick={() => {topOffset-=offsetStep; renderedCoordsCounter--; console.log("top", topOffset)}}>⬆️</button>
        </div>
        <canvas
            width={100} height={100}
            id="my-canvas"
            className="z-40"
            onPointerMove={saveMove}
            onPointerUp={(e) => {
                const buffer = new ArrayBuffer(4);
                const view = new Uint16Array(buffer);
                view[0] = lastSentX
                view[1] = lastSentY
                if (wsState != 1) { return }
                e.preventDefault()
                ws.send(view)
            }}
            onPointerDown={(e) => {
                e.preventDefault()
                const buffer = new ArrayBuffer(4);
                const view = new Uint16Array(buffer);
                view[0] = convertToAbsoluteWidth(e.clientX)
                view[1] = convertToAbsoluteHeight(e.clientY)
                if (wsState != 1) { return }
                ws.send(view)
            }}
        ></canvas>

    </div>)
}

const ZoomBlock = (props: {zoom : number}) => {
    return (
        <button className="p-2 mr-2 bg-blue-300 text-white relative">{roundNum(props.zoom).toFixed(1)}</button>
    )
}