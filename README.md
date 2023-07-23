Tenant configuration is stored in the data directory's `tenants.json` file in the following format:

```json
{
	"app1": {
		"hostnames": {
			"foo.com": "site",
			"app.foo.com": "pwa"
		}
	},
	"app2": {
		"hostnames": {
			"bar.com": "site",
			"app.bar.com": "pwa"
		}
	}
}
```
