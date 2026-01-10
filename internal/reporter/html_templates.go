package reporter

// htmlTemplate is the complete HTML template for the dashboard, assembled from modular components.
// The template is split across multiple files for better maintainability:
//   - html_templates_header.go: HTML head section with CSS styles
//   - html_templates_body.go: Main body structure with all dashboard sections
//   - html_templates_scripts.go: JavaScript code for charts and visualizations
const htmlTemplate = htmlTemplateHeader + htmlTemplateBody + htmlTemplateScripts
