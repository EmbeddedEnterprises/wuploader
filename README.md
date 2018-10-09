# wuploader

wuploader is a file upload utility library for WAMP
This repository uses [nexus](github.com/gammazero/nexus) for WAMP client communication in golang.

wuploader is designed to integrate itself into the [service](github.com/EmbeddedEnterprises/service) library but should work when you use pure nexus too.

---

# How it works

You pass an endpoint which should be a file upload to wuploader and it will register a handler providing high level abstractions for file uploads.

# API Description

`wuploader` defines one procedure with the name given to the API with a polymorphic signature.
Uploading is implemented in a transaction based model, which means that you have to start a transaction first:

`endpoint('start', uploadSize) -> transactionID: uint64`

When starting a transaction, the endpoint may verify the uploader and the upload size and MAY reject the upload, otherwise a valid transaction ID is returned, which has to be specified for subsequent calls. After the transaction has been started, data can be uploaded like this:

`endpoint('data', transactionID, data) -> uploadPos: uint64`

While uploading, the size of the data chunks may be varied, good sizes range from 32kB to 4MB, depending on the connection speed. After all data has been uploaded, the transaction has to be finished like this:

`endpoint('finish', transactionID, args...) -> result: any`

Finish calls the underlying handler with a consistent binary representation of the uploaded content and forwards any arguments/results to the caller/callee.
