export const getErrorPage = (title: string, message: string) => `
<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="UTF-8">
  <meta name="viewport" content="width=device-width, initial-scale=1.0">
  <title>\${title} | l2c</title>
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif; background: #f9fafb; color: #111827; display: flex; align-items: center; justify-content: center; height: 100vh; margin: 0; }
    .container { text-align: center; max-width: 600px; padding: 2.5rem; background: white; border-radius: 12px; box-shadow: 0 4px 6px -1px rgba(0, 0, 0, 0.1), 0 2px 4px -1px rgba(0, 0, 0, 0.06); }
    h1 { margin-top: 0; font-size: 2.5rem; color: #2563eb; margin-bottom: 1rem; }
    p { font-size: 1.1rem; line-height: 1.6; color: #4b5563; margin-bottom: 1.5rem; }
    a { display: inline-block; margin-top: 1rem; padding: 0.75rem 1.5rem; background-color: #2563eb; color: white; text-decoration: none; border-radius: 0.375rem; font-weight: 500; transition: background-color 0.2s; }
    a:hover { background-color: #1d4ed8; }
    code { background: #f1f5f9; padding: 0.2rem 0.4rem; border-radius: 0.25rem; font-size: 0.9em; color: #2563eb; font-weight: bold; }
  </style>
</head>
<body>
  <div class="container">
    <h1>l2c</h1>
    <p>\${message}</p>
    <a href="https://github.com/binodta/l2c" target="_blank" rel="noopener noreferrer">View Documentation</a>
  </div>
</body>
</html>
`;

export const landingPageHTML = getErrorPage("Welcome", "This is a tunneling server powered by <strong>l2c</strong>. To get started, install the CLI and run <code>l2c setup</code>.");
