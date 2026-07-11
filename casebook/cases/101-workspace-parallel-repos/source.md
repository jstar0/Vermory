# Workspace Parallel Repos

Mira is a staff engineer supporting two active repositories for the same company at the same time.

The first repository is `acorn-retail/web-checkout`, a Next.js storefront used by end customers.

In `acorn-retail/web-checkout`, the current live work is on the checkout delivery estimate experience, including a spinner that appears while carrier estimates are loading.

For `web-checkout`, the carrier estimate call currently uses an 800 ms timeout, the preferred experiment flag is `checkout_eta_v2`, and the owner channel is `#checkout-ops`.

The second repository is `acorn-retail/ops-console`, a React admin console used by internal fulfillment agents.

In `acorn-retail/ops-console`, the current live work is on the internal shipment exception review flow used by warehouse support.

For `ops-console`, the exception polling API currently uses a 3-attempt retry budget, the preferred experiment flag is `ops_exception_queue_refresh`, and the owner channel is `#ops-console`.

Both repositories use TypeScript, React, feature flags, and a `ShipmentPanel` component name in different contexts, which creates realistic cross-repo interference pressure.

Right before switching back to `web-checkout`, Mira had another assistant window open on `ops-console` to discuss a stale row problem inside its `ShipmentPanel`.

Mira frequently switches among CLI, IDE chat, and browser-based assistant surfaces, but continuity should stay anchored to the active workspace rather than leak between the two repos by default.
