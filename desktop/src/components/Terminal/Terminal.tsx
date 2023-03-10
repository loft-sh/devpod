import React from "react"
import * as XTerm from "xterm"
import { FitAddon } from "xterm-addon-fit"

export interface TerminalProps {
    value?: string
    className?: string
    cursorBlink?: boolean
    disableStdin?: boolean

    addons?: XTerm.ITerminalAddon[]

    height?: number
    width?: number
}

export class Terminal extends React.PureComponent<TerminalProps> {
    private term: XTerm.Terminal
    private termFit: FitAddon
    private ref: React.RefObject<HTMLDivElement>

    constructor(props: TerminalProps) {
        super(props)

        this.ref = React.createRef()
        this.term = new XTerm.Terminal({
            // We need this setting to automatically convert \n -> \r\n
            convertEol: true,
            fontSize: 12,
            scrollback: 25000,
            cursorBlink: this.props.cursorBlink != null ? this.props.cursorBlink : false,
            disableStdin: this.props.disableStdin != null ? this.props.disableStdin : true,
            theme: {
                background: "#263544",
                foreground: "#AFC6D2",
            },
        })

        this.termFit = new FitAddon()
        this.term.loadAddon(this.termFit)

        const { addons = [] } = this.props
        for (const addon of addons) {
            this.term.loadAddon(addon)
        }

        this.term.onKey((key) => {
            if (this.term.hasSelection() && key.domEvent.ctrlKey && key.domEvent.key === "c") {
                document.execCommand("copy")
            }
        })
    }

    clear = () => {
        this.term.clear()
    }

    write = (data: string) => {
        this.term.write(data)
        this.updateDimensions()
    }

    writeln = (data: string) => {
        this.term.writeln(data)
        this.updateDimensions()
    }

    updateDimensions = () => {
        this.termFit.fit()
    }

    componentDidUpdate() {
        this.updateDimensions()
    }

    componentDidMount() {
        window.addEventListener("resize", this.updateDimensions, true)

        this.term.open(this.ref.current!)
        this.updateDimensions()
    }

    componentWillUnmount() {
        window.removeEventListener("resize", this.updateDimensions, true)
        this.term.dispose()
        this.termFit.dispose()
        const { addons = [] } = this.props
        for (const addon of addons) {
            addon.dispose()
        }
    }

    render() {
        const classnames = [""]; //[styles.terminalWrapper]
        if (this.props.className) {
            classnames.push(this.props.className)
        }

        return (
            <div
                className={classnames.join(" ")}
                style={{
                    display: "flex",
                    width: this.props.width ? `${this.props.width}px` : undefined,
                    height: this.props.height ? `${this.props.height}px` : undefined,
                }}>
                <div className={""} ref={this.ref} />
            </div>
        )
    }
}
