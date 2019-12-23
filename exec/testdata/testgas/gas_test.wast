(module
  (func (export "gas-add") (param $x i32) (param $y i32) (result i32)
   local.get $x
   local.get $y
    i32.add
  )
  ;;local.get + local.get + i32.add + function.end = 4

  (func (export "gas-empty-block")
      (block)
      (block $l)
  )
  ;;block + end + block + end + function.end = 5

  (func (export "gas-type-i32-value-br") (result i32)
    (block (result i32)
        i32.const 1
        br 0
        i32.ctz
        i32.const 2
        i32.add
       )
    ;;(block (result i32) (i32.ctz (br 0 (i32.const 1))))
  )
  ;; block + i32.const + br + function.end= 4

  (func (export "gas-as-br-value") (result i32)
      (block (result i32)
         i32.const 1
         i32.const 2
         br_if 0
         br 0
      )
  )
  ;;block + i32.const + i32.const + br_if + br = 5

  (func (export "gas-type-i32-value-br-table") (result i32)
    (block (result i32)
     i32.const 0
     i32.const 1
     br_table 0 0
     i32.ctz
    )
  )
  ;;block + i32.const + i32.const + br_table + i32.ctz= 5

  (global $x (mut i32) (i32.const -12))
  (func (export "gas-set-x") (param i32)
   local.get 0
   global.set $x
  )
  ;;local.get + global.set + function.end = 3

  (func (export "gas-empty-if") (param i32)
   (if (local.get 0) (then))
  )
  ;;if + end + local.get = 3

  (func $i32-i64 (param i32 i64) (result i64) (local.get 1))
  (func (export "gas-type-second-i64") (result i64)
      (call $i32-i64 (i32.const 32) (i64.const 64))
  )
  ;;call + i32.const + return + local.get + i64.const + return = 6

  (memory 3)
  (func (export "gas-empty-loop")
      (loop $l)
  )
  ;;loop + end + end = 3

  (func (export "gas-singular-loop") (result i32)
      (loop (result i32) (i32.const 7))
  )
  ;;loop + end + i32.const + end = 4

  (func (export "gas-load_at_zero") (result i32)
   i32.const 0
   i32.load
  )
  ;; i32.load + i32.const + return = 3

  (func (export "gas-select_i32") (param $lhs i32) (param $rhs i32) (param $cond i32) (result i32)
     (select (local.get $lhs) (local.get $rhs) (local.get $cond)))
  ;; local.get + local.get + local.get + select +return = 5


   (func (export "gas-as-func-first") (result i32)
      (nop) (i32.const 1)
    )
    ;;nop + i32.const + return = 3

  ;;switch
  ;;i32.const+local.set+block+switch+block+block+block+block+block+block+block+block+block+return+local.get+return = 16
  (func (export "gas-stmt") (param $i i32) (result i32)
      (local $j i32)
      (local.set $j (i32.const 100))
      (block $switch
        (block $7
          (block $default
            (block $6
              (block $5
                (block $4
                  (block $3
                    (block $2
                      (block $1
                        (block $0
                          (br_table $0 $1 $2 $3 $4 $5 $6 $7 $default
                            (local.get $i)
                          )
                        ) ;; 0
                        (return (local.get $i))
                      ) ;; 1
                      (nop)
                      ;; fallthrough
                    ) ;; 2
                    ;; fallthrough
                  ) ;; 3
                  (local.set $j (i32.sub (i32.const 0) (local.get $i)))
                  (br $switch)
                ) ;; 4
                (br $switch)
              ) ;; 5
              (local.set $j (i32.const 101))
              (br $switch)
            ) ;; 6
            (local.set $j (i32.const 101))
            ;; fallthrough
          ) ;; default
          (local.set $j (i32.const 102))
        ) ;; 7
        ;; fallthrough
      )
      (return (local.get $j))
    )
)