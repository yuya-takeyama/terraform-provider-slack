# terraform-provider-slack

## Pointing the provider at a proxy (`api_url` / `SLACK_API_URL`)

Every Slack API call the provider makes, including the `apps.manifest.*` calls behind `slack_app`, goes to `https://slack.com/api/` by default. Set the provider's `api_url` attribute, or the `SLACK_API_URL` environment variable when the attribute is unset, to send them somewhere else. The intended use is a CI setup where the plan job never holds a real token: run a proxy in front of Slack that only forwards read methods such as `apps.manifest.export` and `conversations.info` and injects the real tokens itself, then point the plan job at that proxy with `SLACK_API_URL` while the apply job keeps talking to `slack.com` with the real tokens. The value must be an `http` or `https` URL; the trailing slash may be omitted.
