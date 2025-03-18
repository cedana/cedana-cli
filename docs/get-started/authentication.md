## Authentication

Create an account at [auth.cedana.com](https://auth.cedana.com).

Once logged in, you'll be redirected to an account management page, where you can create an API key. If you've logged in via your organization's email address; you'll be able to create and manage organization-wide API keys as well.

Once you have obtained an API key, export the following variables: 

```
export CEDANA_URL="https://<org-name>.cedana.ai/v1"
export CEDANA_AUTH_TOKEN=<Your auth token from https://auth.cedana.com>
```

You should have received a unique URL for your organization.
