# Proposal
Fix repeated neural queue overflow across timeline resets. Old output packets occupy the bounded queue until normal delayed output consumption; repeated resets during pre-roll can fill it and make every retry fail before consumption starts.
