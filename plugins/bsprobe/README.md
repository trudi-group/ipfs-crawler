# Bitswap Probing Plugin

A plugin to probe nodes for content via Bitswap.

## Configuration

```yaml
- name: "bitswap-probe"
  options:
    # A list of CIDs to ask for
    cids:
      # CID of the IPFS logo
      - "QmY7Yh4UquoXHLPFo2XbhXkhBvFoPwmQUSa92pxnxjQuPU"

    # The timeout to use for requests
    request_timeout: "5s"

    # The period of time to wait for replies
    response_period: "30s"
```

## Results

```json
"bitswap-probe": {
  "error": null,
  "result": {
    "error": null,
    "haves": null,
    "dont_haves": [
      {
        "/": "QmY7Yh4UquoXHLPFo2XbhXkhBvFoPwmQUSa92pxnxjQuPU"
      }
    ],
    "blocks": null,
    "no_response": null
  }
}
```

See also the documented `ProbeResult` type.