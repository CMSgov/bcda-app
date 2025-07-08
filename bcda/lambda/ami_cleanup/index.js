const AWS = require('aws-sdk');
const querystring = require('querystring');
const https = require('https');

const ec2 = new AWS.EC2({
    apiVersion: '2016-11-15'
});

const sixty_days_ago = new Date().getTime() - (60 * 24 * 60 * 60 * 1000);

const params = {
    Filters: [{
        Name: 'tag:Name',
        Values: ['bcda-*', 'dpc-*']
    }],
};

const send_slack_message = (message) => {
    const url = process.env.SLACK_WEBHOOK_URL;

    const postData = JSON.stringify({
        text: message
    });

    const options = {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'Content-Length': Buffer.byteLength(postData)
        }
    };

    const req = https.request(url, options, (res) => {
        console.log(`STATUS: ${res.statusCode}`);
        console.log(`HEADERS: ${JSON.stringify(res.headers)}`);
        res.setEncoding('utf8');
        res.on('data', (chunk) => {
            console.log(`BODY: ${chunk}`);
        });
        res.on('end', () => {
            console.log('No more data in response.');
        });
    });

    req.on('error', (e) => {
        console.error(`problem with request: ${e.message}`);
    });

    req.write(postData);
    req.end();
};

const handle_error = (err) => {
    console.log(err, err.stack);
};

const deregister_old_amis = (data) => {
    if (!data || !data.Images) {
        console.warn("Unexpected data value");
        throw new Error("Unexpected data value");
    };

    const to_deregister = data.Images
        .filter(image => {
            if (!image.CreationDate) return false;
            const creationTime = new Date(image.CreationDate).getTime();
            return !isNaN(creationTime) && creationTime < sixty_days_ago;
        });

    console.log(`Deregistering ${to_deregister.length} images`);

    const promises_to_deregister_images = to_deregister.map((image) => ec2.deregisterImage({
        ImageId: image.ImageId,
        DryRun: false
    }).promise());

    return Promise.all(promises_to_deregister_images);
};

const delete_old_snapshots = (data) => {
    if (!data || !data.Snapshots) {
        throw new Error("Unexpected data value");
    }

    const to_delete = data.Snapshots.filter(snapshot => new Date(snapshot.StartTime).getTime() < sixty_days_ago);

    console.log(`Deleting ${to_delete.length} snapshots`);

    const promises_to_delete_snapshots = to_delete.map((snapshot) => ec2.deleteSnapshot({
        SnapshotId: snapshot.SnapshotId,
        DryRun: false
    }).promise());

    return Promise.all(promises_to_delete_snapshots);
};

const cleanup_amis = () => {
    return ec2.describeImages(params).promise()
        .then(deregister_old_amis, handle_error)
        .then((results) => {
            console.log(`Deregistered ${results.length} images.`);
            send_slack_message(`AMI Cleanup: Deregistered ${results.length} AMIs older than sixty days.`);
        })
        .catch((err) => {
            console.log(`Error: Deregistering old AMIs failed: ${err}`);
        });
};

const cleanup_snapshots = () => {
    return ec2.describeSnapshots(params).promise()
        .then(delete_old_snapshots, handle_error)
        .then((results) => {
            console.log(`Deleted ${results.length} snapshots.`);
            send_slack_message(`AMI Cleanup: Deleted ${results.length} snapshots older than sixty days.`);
        })
        .catch((err) => {
            console.log(`Error: Deleting old snapshots failed: ${err}`);
        })
};

exports.handler = (event, context, callback) => {
    cleanup_amis().then(cleanup_snapshots);
};
