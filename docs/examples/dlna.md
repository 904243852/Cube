# Using DLNA to cast a video on device in LAN

1. Create a controller with url `/dlna`.
    ```typescript
    export default function (ctx: ServiceContext): ServiceResponse | Uint8Array | any {
        const videoUri = "https://s.xlzys.com/play/9b64Eq9e/index.m3u8" // m3u8 or mp4

        // 1. Search for a device using SSDP
        // create a virtual udp connection using a random port
        const ssdpc = $native("socket")("udp").listen(0)
        // send a M-Search broadcast packet to 239.255.255.250:1900 (the source port is exactly what "ssdpc" used)
        ssdpc.write([
            "M-SEARCH * HTTP/1.1",
            "HOST: 239.255.255.250:1900", // broadcast address (239.255.255.250:1900 is the default ip and port of SSDP broadcast)
            `MAN: "ssdp:discover"`,
            "MX: 3", // maximum waiting time
            "ST: urn:schemas-upnp-org:service:AVTransport:1", // service type
            "",
        ].join("\r\n"), "239.255.255.250", 1900)
        // wait for device response
        const devices = [...new Array(1)].reduce((p) => {
            const b = Buffer.from(ssdpc.read()).toString() // listen on the port that "ssdpc" used and read bytes
            return [
                ...p,
                {
                    location: b.match(/LOCATION:\s([^\r\n]*)/i)[1],
                    server: b.match(/SERVER:\s([^\r\n]*)/i)[1],
                },
            ]
        }, [])

        // 2. Query controlURL from a device
        const [{ location }] = devices // here we just use the first device for example
        const host = location.match(/^https?:\/\/[^\/]+/)[0]
        const service = $native("xml")($native("http")().request("get", location).data.toString()) // request location and parse it into an XML dom
            .find("//service").map(i => {
                return {
                    serviceType: i.findOne("servicetype").innerText(),
                    serviceId: i.findOne("serviceid").innerText(),
                    controlURL: i.findOne("controlurl").innerText(),
                }
            })
            .filter(i => i.serviceType === "urn:schemas-upnp-org:service:AVTransport:1")
            .pop()
        if (!service) {
            throw new Error("service not found: AVTransport")
        }

        // 3. SetAVTransportURI
        return $native("http")().request("post",
            host + service.controlURL,
            {
                "Content-Type": `text/xml;charset="utf-8"`,
                "SOAPACTION": `"urn:schemas-upnp-org:service:AVTransport:1#SetAVTransportURI"`, // SOAPACTION: "${serviceType}#${action}"
            },
            `<?xml version="1.0" encoding="utf-8"?>
    <s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/">
        <s:Body>
            <u:SetAVTransportURI xmlns:u="urn:schemas-upnp-org:service:AVTransport:1">
                <InstanceID>0</InstanceID>
                <CurrentURI>${videoUri}</CurrentURI>
                <CurrentURIMetaData></CurrentURIMetaData>
            </u:SetAVTransportURI>
        </s:Body>
    </s:Envelope>`
        ).data.toString()
    }
    ```

2. Now you can call `http://127.0.0.1:8090/service/dlna` to case a video on device in LAN.
