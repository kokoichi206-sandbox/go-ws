## 文字化けのようになってしまう

``` sh
$ go run main.go -name=pien
hello im minami
hello im ore
hello im ore
ɉ��     L��`"�����j����������������Ί���87��H^�����$E.�A)B�,C�K7K
```

全ての frameReader を読み取ってなかったことが原因？

読み取らないとなぜそうなってしまうのか。
