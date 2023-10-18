# -*- coding: utf-8 -*-

# Used to Wrap Text
import textwrap

# Download the files from https://www.cs.cmu.edu/afs/cs/project/ai-repository/ai/areas/nlp/corpora/names/
import os

os.system('wget https://www.cs.cmu.edu/afs/cs/project/ai-repository/ai/areas/nlp/corpora/names/female.txt https://www.cs.cmu.edu/afs/cs/project/ai-repository/ai/areas/nlp/corpora/names/male.txt https://www.cs.cmu.edu/afs/cs/project/ai-repository/ai/areas/nlp/corpora/names/other/family.txt https://www.cs.cmu.edu/afs/cs/project/ai-repository/ai/areas/nlp/corpora/names/other/names.txt')

# Read an process family.txt
with open("/content/family.txt", "r+") as familyfile:
  familynames = [ line.strip().replace(" ", "-") for line in familyfile ]
familynames = list(set(familynames))

# Read an process female.txt
with open("/content/female.txt", "r+") as femalefile:
  femalenames = [ line.strip().replace(" ", "-") for line in femalefile ]
femalenames = list(set(femalenames[6:]))

# Read an process male.txt
with open("/content/male.txt", "r+") as malefile:
  malenames = [ line.strip().replace(" ", "-") for line in malefile ]
malenames = list(set(malenames[6:]))

# Read an process names.txt
with open("/content/names.txt", "r+") as namesfile:
  allnames = [ line.strip().replace(" ", "-") for line in namesfile ]
allnames = list(set(allnames))

print(f"""Number of female names is {len(femalenames)}\n
Number of male names is {len(malenames)}\n
Number of family names is {len(familynames)}\n
Number of general names is {len(allnames)}\n""")

family_var = "FamilyNames = []string{\"" + "\", \"".join(familynames) + "\"}"
wrapped_family_text = textwrap.wrap(
  family_var, width=100, break_on_hyphens=False, break_long_words=False
)

female_var = "FemaleNames = []string{\"" + "\", \"".join(femalenames) + "\"}"
wrapped_female_text = textwrap.wrap(
  female_var, width=100, break_on_hyphens=False, break_long_words=False
)

male_var = "MaleNames = []string{\"" + "\", \"".join(malenames) + "\"}"
wrapped_male_text = textwrap.wrap(
  male_var, width=100, break_on_hyphens=False, break_long_words=False
)

general_var = "GeneralNames = []string{\"" + "\", \"".join(allnames) + "\"}"
wrapped_general_text = textwrap.wrap(
  general_var, width=100, break_on_hyphens=False, break_long_words=False
)

with open("names.go", "w") as f:
  f.write("package namegenerator\n")
  f.write("var (\n")
  f.write("// FamilyNames is a list of family names\n")
  for line in wrapped_family_text:
        f.write(line + "\n")
  f.write("\n// FemaleNames is a list of female names\n")
  for line in wrapped_female_text:
        f.write(line + "\n")
  f.write("\n// MaleNames is a list of male names\n")
  for line in wrapped_male_text:
        f.write(line + "\n")
  f.write("\n// GeneralNames is a list of general names\n")
  for line in wrapped_general_text:
        f.write(line + "\n")
  f.write(")")
