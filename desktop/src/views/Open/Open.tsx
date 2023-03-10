import React, {MutableRefObject, useEffect, useRef} from "react";
import {Terminal} from "../../components/Terminal/Terminal";
import { Command } from '@tauri-apps/api/shell';

export function Open() {
    const terminalRef = useRef<Terminal>(null);
    useEffect(() => {
        (async () => {
            const command = Command.sidecar('bin/devpod', ["up"]);
            command.on('close', data => {
                console.log(`command finished with code ${data.code} and signal ${data.signal}`)
            });
            command.on('error', error => console.error(`command error: "${error}"`));
            command.stdout.on('data', line => terminalRef.current?.write(line));
            command.stderr.on('data', line => terminalRef.current?.write(line));
            await command.spawn();
        })()
    }, [])

    return <Terminal ref={terminalRef} />
}