; Simple function call: foo(args)
(call_expression
  function: (identifier) @callee.name)

; Method call via . or ->: obj.method() or obj->method()
(call_expression
  function: (field_expression
    field: (field_identifier) @callee.name))
