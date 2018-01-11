
.PHONY: all clean

INPUTS=input-diff-without-color.txt input-diff-with-color.txt
EXPECTED=expected-without-color.txt expected-with-color.txt

all: $(INPUTS) $(EXPECTED)

input-diff-without-color.txt: old.txt new.txt
	git diff --no-index --color=never $^ > $@ &2> /dev/null

input-diff-with-color.txt: old.txt new.txt
	git diff --no-index --color=always $^ > $@ &2> /dev/null

expected-%.txt: input-diff-%.txt
	gdhighlight < $< > $@
	@echo "updated $@"

clean:
	rm -f $(INPUTS) $(EXPECTED)
