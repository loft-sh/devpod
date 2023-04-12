# systray

Go package for Windows Systray icon, menu and notifications.

## Dependencies

No other dependencies except Go standard library.

## Building

If you want to package icon files and other resources into binary **rsrc** tool is recommended:

	rsrc -manifest app.manifest -ico=app.ico,application_edit.ico,application_error.ico -o rsrc.syso

Here app.manifest is XML file in format:
```
<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<assembly xmlns="urn:schemas-microsoft-com:asm.v1" manifestVersion="1.0">
    <assemblyIdentity version="1.0.0.0" processorArchitecture="*" name="App" type="win32"/>
    <dependency>
        <dependentAssembly>
            <assemblyIdentity type="win32" name="Microsoft.Windows.Common-Controls" version="6.0.0.0" processorArchitecture="*" publicKeyToken="6595b64144ccf1df" language="*"/>
        </dependentAssembly>
    </dependency>
</assembly>
```

Most Windows applications do not display command prompt. Build your Go project with flag to indicate that it is Windows GUI binary:

	go build -ldflags="-H windowsgui"

## Samples

Best way to learn how to use the library is to look at the included **example** project.

![Hello World](example/screenshot.png)

Use **release.bat** to build it.

## Caveats

Package is designed to run as standalone GUI application. That means it runs its own Windows message loop.
This can have unexpected side effects if you try to combine with other UI packages that also run they
own message loops.

## Credits

This library is built based on

[xilp/systray](https://github.com/xilp/systray)

Constant definitions and syscall declarations have been reused from that package.
