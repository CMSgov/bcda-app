// Uses Cloudfront 1.0, since version is locked to v4.31.0 in versions.tf

var redirects = {
    '/guide': '/api-documentation',
    '/build': '/api-documentation/access-claims-data',
    '/data': '/bcda-data',
    '/updates': '/announcements',
    '/partial': '/bcda-data/partially-adjudicated-claims-data',
}

function handler(event) {
    var request = event.request;

    // Handle Site Redirects (ex. https://github.com/aws-samples/amazon-cloudfront-functions/blob/main/redirect-based-on-country/index.js)
    var uri = request.uri
    if (uri.endsWith('.html')) {
        uri = uri.slice(0, -5);
    }

    if (redirects[uri]) {
        var response = {
            statusCode: 301,
            statusDescription: 'Moved Permanently',
            headers: {
                location: {
                    value: 'https://' + request.headers.host.value + redirects[uri]
                }
            }
        }
        return response;
    }

    // Handle "Cool URIs" (ex. https://github.com/aws-samples/amazon-cloudfront-functions/blob/main/url-rewrite-single-page-apps/index.js)
    // Rewrite to index.html in a directory when requesting its root
    if (request.uri.endsWith('/')) {
        request.uri += 'index.html';
    }
    // Rewrite with added ".html" when there's no file extension
    else if (!request.uri.includes('.')) {
        request.uri += '.html';
    }

    return request
}
