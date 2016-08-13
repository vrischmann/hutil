hutil
======

introduction
------------

hutil is a set of helpers to work with `net/http`. It's limited by design, based on what I need the most in my projects.

chain
-----

The most useful struct is `hutil.Chain`. It is used to chain middleware together. Look at the documentation [here](https://godoc.org/github.com/vrischmann/hutil#Chain) to see how to use it.

middlewares
-----------

Only one middleware is provided for now, a logging middleware. Look at the documentation [here](https://godoc.org/github.com/vrischmann/hutil#Chain) to see how to use it.

logging
-------

You can get a logging middlware this way:

    m := hutil.NewLoggingMiddleware(nil)

You can configure the middleware by providing a `hutil.LoggingOptions` struct:

    m := hutil.NewLoggingMiddleware(&hutil.LoggingOptions{
        WithExecutionTime: true,
    })

Look at the documentation [here](https://godoc.org/github.com/vrischmann/hutil#LoggingOptions) for all available options.
