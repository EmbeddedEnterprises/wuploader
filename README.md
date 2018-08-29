# wuploader

wuploader is a file upload utility library for WAMP
This repository uses [nexus](github.com/gammazero/nexus) for WAMP client communication in golang.

wuploader is designed to integrate itself into the [service](github.com/EmbeddedEnterprises/service) library but should work when you use pure nexus too.

---

# How it works

You pass an endpoint which should be a file upload to wuploader and it will register a handler providing high level abstractions for file uploads.
