# About
**Burp2Caido** is a tool that allows you to migrate exported Burpsuite HTTP history files into Caido. If you have any questions, reach out to me on [Twitter](https://twitter.com/monkehack).
# Installation
`go install -v github.com/projectmonke/burp2caido@latest`
# Usage
The tool takes two command-line arguments.
- `--burp` specifies where the HTTP history XML file is.
- `--caido` specifies where the Caido project folder is.

1. Open Burpsuite, and select your project. Navigate to the Proxy tab, and highlight the requests you want to export, right click and select "Save Items". Name your XML file something memorable and save it, and remember the path.
2. Create a new, empty Caido project. Navigate to the **Workspace** tab and click the three dots to the right, and select "Copy Path".
3. Run the following command, replacing the placeholders with the two paths above: `burp2caido --burp <burpsuite path> --caido <caido path>`
4. If you switch out of your new Caido project and switch back to it, you should see the traffic in the HTTP History tab in Caido.

# Warning
This tool inserts HTTP data directly into the Caido project. Running it multiple times WILL add the requests again each time. Run it once!

# Disclaimer
If something is broken, fork it yourself and fix it. I'm usually pretty busy. I do intend on expanding the capabilities of this tool, but I'm making no promises about when that'll happen. However, feel free to open issues and I'll try to answer them. Also, sometimes Caido just deletes the imported history or it won't be visible, and I have no idea why, so please just try again if that happens.
