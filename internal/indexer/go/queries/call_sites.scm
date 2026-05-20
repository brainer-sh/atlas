; Simple function call: foo(args)
(call_expression
  function: (identifier) @callee.name)

; Method or package call: x.Foo(args)
(call_expression
  function: (selector_expression
    field: (field_identifier) @callee.name))
