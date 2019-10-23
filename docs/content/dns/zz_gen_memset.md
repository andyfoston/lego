---
title: "Memset"
date: 2019-03-03T16:39:46+01:00
draft: false
slug: memset
---

<!-- THIS DOCUMENTATION IS AUTO-GENERATED. PLEASE DO NOT EDIT. -->
<!-- providers/dns/memset/memset.toml -->
<!-- THIS DOCUMENTATION IS AUTO-GENERATED. PLEASE DO NOT EDIT. -->

Since: v3.2.0

Configuration for [Memset](https://www.memset.com/docs/managing-your-server/dns/).


<!--more-->

- Code: `memset`

{{% notice note %}}
_Please contribute by adding a CLI example._
{{% /notice %}}




## Credentials

| Environment Variable Name | Description |
|-----------------------|-------------|
| `MEMSET_AUTH_TOKEN` | API Key |

The environment variable names can be suffixed by `_FILE` to reference a file instead of a value.
More information [here](/lego/dns/#configuration-and-credentials).


## Additional Configuration

| Environment Variable Name | Description |
|--------------------------------|-------------|
| `MEMSET_HTTP_TIMEOUT` | API request timeout |
| `MEMSET_POLLING_INTERVAL` | Time between DNS propagation check |
| `MEMSET_PROPAGATION_TIMEOUT` | Maximum waiting time for DNS propagation |
| `MEMSET_TTL` | The TTL of the TXT record used for the DNS challenge |

The environment variable names can be suffixed by `_FILE` to reference a file instead of a value.
More information [here](/lego/dns/#configuration-and-credentials).




## More information

- [API documentation](https://www.memset.com/apidocs/methods_dns.html)

<!-- THIS DOCUMENTATION IS AUTO-GENERATED. PLEASE DO NOT EDIT. -->
<!-- providers/dns/memset/memset.toml -->
<!-- THIS DOCUMENTATION IS AUTO-GENERATED. PLEASE DO NOT EDIT. -->
