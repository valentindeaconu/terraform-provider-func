data "func" "sum" {
  id = "sum"

  inputs = [100, 100]
}

data "func" "create_object" {
  id = "create_object"

  inputs = {
    age  = 35
    name = "Bob"
  }
}

output "results" {
  value = {
    data_sum           = data.func.sum.result                        // 200
    data_create_object = data.func.create_object.result              // { name = "Bob", age = 35 }
    sum                = provider::func::sum(10, 5)                  // 15
    min                = provider::func::min(10, 5)                  // 5
    concat             = provider::func::concat("Hello, ", "world!") // "Hello, world!"
    extend             = provider::func::extend([1, 1, 2], [3])      // [1, 1, 2, 3]
    create_object      = provider::func::create_object("John", 35)   // { name = "John", age = 35 }
  }
}
